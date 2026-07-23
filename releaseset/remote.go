package releaseset

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const maxRemoteMetadataSize = 1 << 20

type commitResponse struct {
	SHA string `json:"sha"`
}

type workflowResponse struct {
	HeadSHA    string `json:"head_sha"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

// VerifyRemote checks public tags, schemas, workflow evidence, and artifacts.
func VerifyRemote(ctx context.Context, client HTTPClient, set Set) error {
	if client == nil {
		return fmt.Errorf("release set: HTTP client is required")
	}
	if err := set.Validate(); err != nil {
		return err
	}
	for _, component := range set.Components {
		if err := verifyComponent(ctx, client, component); err != nil {
			return err
		}
	}
	for _, schema := range set.Schemas {
		if err := verifySchema(ctx, client, schema); err != nil {
			return err
		}
	}
	if err := verifyWorkflow(ctx, client, set.Evidence); err != nil {
		return err
	}
	return VerifyArtifacts(ctx, client, set)
}

func verifyComponent(ctx context.Context, client HTTPClient, component Component) error {
	endpoint := fmt.Sprintf(
		"https://api.github.com/repos/%s/commits/%s",
		component.Repository,
		url.PathEscape(component.Version),
	)
	var response commitResponse
	if err := getJSON(ctx, client, endpoint, &response); err != nil {
		return fmt.Errorf("release set: component %q: %w", component.Name, err)
	}
	if response.SHA != component.Commit {
		return fmt.Errorf("release set: component %q tag resolves to %q, want %q", component.Name, response.SHA, component.Commit)
	}
	return nil
}

func verifySchema(ctx context.Context, client HTTPClient, schema Schema) error {
	request, err := newRequest(ctx, schema.URL)
	if err != nil {
		return fmt.Errorf("release set: schema %q: %w", schema.Name, err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("release set: schema %q: download: %w", schema.Name, err)
	}
	defer closeBody(response.Body)

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("release set: schema %q: download returned %s", schema.Name, response.Status)
	}
	hash := sha256.New()
	size, err := io.Copy(hash, io.LimitReader(response.Body, maxDocumentSize+1))
	if err != nil {
		return fmt.Errorf("release set: schema %q: read: %w", schema.Name, err)
	}
	if size > maxDocumentSize {
		return fmt.Errorf("release set: schema %q is too large", schema.Name)
	}
	want, _ := checksumBytes(schema.Checksum)
	if subtle.ConstantTimeCompare(hash.Sum(nil), want) != 1 {
		return fmt.Errorf("release set: schema %q: checksum mismatch", schema.Name)
	}
	return nil
}

func verifyWorkflow(ctx context.Context, client HTTPClient, evidence Evidence) error {
	parsed, _ := url.Parse(evidence.Workflow)
	parts := strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/")
	endpoint := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/actions/runs/%s",
		parts[0],
		parts[1],
		parts[4],
	)
	var response workflowResponse
	if err := getJSON(ctx, client, endpoint, &response); err != nil {
		return fmt.Errorf("release set: evidence: %w", err)
	}
	if response.HeadSHA != evidence.Commit {
		return fmt.Errorf("release set: evidence workflow commit is %q, want %q", response.HeadSHA, evidence.Commit)
	}
	if response.Status != "completed" || response.Conclusion != "success" {
		return fmt.Errorf(
			"release set: evidence workflow is %s with conclusion %s",
			response.Status,
			response.Conclusion,
		)
	}
	return nil
}

func getJSON(ctx context.Context, client HTTPClient, endpoint string, value any) error {
	request, err := newRequest(ctx, endpoint)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer closeBody(response.Body)

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("request returned %s", response.Status)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, maxRemoteMetadataSize+1))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if len(content) > maxRemoteMetadataSize {
		return fmt.Errorf("response is too large")
	}
	if err := json.Unmarshal(content, value); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func newRequest(ctx context.Context, endpoint string) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "pawn-actions-release-set")
	return request, nil
}

func closeBody(body io.ReadCloser) {
	_ = body.Close()
}

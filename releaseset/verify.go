package releaseset

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"io"
	"net/http"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

func VerifyArtifacts(ctx context.Context, client HTTPClient, set Set) error {
	if client == nil {
		return fmt.Errorf("release set: HTTP client is required")
	}
	if err := set.Validate(); err != nil {
		return err
	}
	for _, component := range set.Components {
		for _, artifact := range component.Artifacts {
			if err := verifyArtifact(ctx, client, component.Name, artifact); err != nil {
				return err
			}
			if artifact.SBOM != nil {
				if err := verifyAsset(ctx, client, component.Name+" SBOM", artifact.SBOM.URL, artifact.SBOM.Size, artifact.SBOM.Checksum); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func verifyArtifact(ctx context.Context, client HTTPClient, component string, artifact Artifact) error {
	return verifyAsset(ctx, client, component, artifact.URL, artifact.Size, artifact.Checksum)
}

func verifyAsset(ctx context.Context, client HTTPClient, component, url string, size int64, checksum string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("release set: %s: create request: %w", component, err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("release set: %s: download: %w", component, err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("release set: %s: download returned %s", component, response.Status)
	}
	if response.ContentLength >= 0 && response.ContentLength != size {
		return fmt.Errorf("release set: %s: artifact size is %d, want %d", component, response.ContentLength, size)
	}

	hash := sha256.New()
	actualSize, err := io.Copy(hash, io.LimitReader(response.Body, size+1))
	if err != nil {
		return fmt.Errorf("release set: %s: read artifact: %w", component, err)
	}
	if actualSize != size {
		return fmt.Errorf("release set: %s: artifact size is %d, want %d", component, actualSize, size)
	}
	want, _ := checksumBytes(checksum)
	if subtle.ConstantTimeCompare(hash.Sum(nil), want) != 1 {
		return fmt.Errorf("release set: %s: checksum mismatch", component)
	}
	return nil
}

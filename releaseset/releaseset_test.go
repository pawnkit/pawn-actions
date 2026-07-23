package releaseset

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDecodeValidSet(t *testing.T) {
	t.Parallel()

	content, err := json.Marshal(validSet([]byte("archive")))
	if err != nil {
		t.Fatal(err)
	}
	set, err := Decode(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if set.ID != "preview-2026-07-23" {
		t.Fatalf("ID = %q", set.ID)
	}
}

func TestDecodeRejectsUnknownAndTrailingData(t *testing.T) {
	t.Parallel()

	for _, content := range []string{
		`{"schemaVersion":1,"unknown":true}`,
		`{} {}`,
	} {
		if _, err := Decode(strings.NewReader(content)); err == nil {
			t.Fatalf("Decode(%q) succeeded", content)
		}
	}
}

func TestDecodeRejectsOversizedDocument(t *testing.T) {
	t.Parallel()

	content := strings.Repeat(" ", maxDocumentSize+1)
	if _, err := Decode(strings.NewReader(content)); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("Decode error = %v", err)
	}
}

func TestValidateRejectsDuplicateComponent(t *testing.T) {
	t.Parallel()

	set := validSet([]byte("archive"))
	set.Components = append(set.Components, set.Components[0])
	if err := set.Validate(); err == nil || !strings.Contains(err.Error(), "duplicate component") {
		t.Fatalf("Validate error = %v", err)
	}
}

func TestValidateRejectsUntestedArtifactTarget(t *testing.T) {
	t.Parallel()

	set := validSet([]byte("archive"))
	set.Components[0].Artifacts[0].Target = "windows-amd64"
	if err := set.Validate(); err == nil || !strings.Contains(err.Error(), "untested target") {
		t.Fatalf("Validate error = %v", err)
	}
}

func TestValidateRejectsMutableVersion(t *testing.T) {
	t.Parallel()

	set := validSet([]byte("archive"))
	set.Components[0].Version = "main"
	if err := set.Validate(); err == nil || !strings.Contains(err.Error(), "invalid component") {
		t.Fatalf("Validate error = %v", err)
	}
}

func TestValidateRequiresMatchingEvidence(t *testing.T) {
	t.Parallel()

	set := validSet([]byte("archive"))
	set.Evidence.Commit = strings.Repeat("9", 40)
	if err := set.Validate(); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("Validate error = %v", err)
	}
}

func TestVerifyArtifacts(t *testing.T) {
	t.Parallel()

	content := []byte("archive")
	set := validSet(content)
	client := staticClient{response: &http.Response{
		StatusCode:    http.StatusOK,
		Status:        "200 OK",
		ContentLength: int64(len(content)),
		Body:          io.NopCloser(bytes.NewReader(content)),
	}}
	if err := VerifyArtifacts(t.Context(), client, set); err != nil {
		t.Fatalf("VerifyArtifacts: %v", err)
	}
}

func TestVerifyArtifactsRejectsChecksumMismatch(t *testing.T) {
	t.Parallel()

	set := validSet([]byte("wanted"))
	content := []byte("actual")
	client := staticClient{response: &http.Response{
		StatusCode:    http.StatusOK,
		Status:        "200 OK",
		ContentLength: int64(len(content)),
		Body:          io.NopCloser(bytes.NewReader(content)),
	}}
	if err := VerifyArtifacts(t.Context(), client, set); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("VerifyArtifacts error = %v", err)
	}
}

func TestVerifyArtifactsRejectsMissingArtifact(t *testing.T) {
	t.Parallel()

	set := validSet([]byte("wanted"))
	client := staticClient{response: &http.Response{
		StatusCode: http.StatusNotFound,
		Status:     "404 Not Found",
		Body:       io.NopCloser(strings.NewReader("")),
	}}
	if err := VerifyArtifacts(t.Context(), client, set); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("VerifyArtifacts error = %v", err)
	}
}

func TestValidateGoMod(t *testing.T) {
	t.Parallel()

	valid := []byte("module example.com/project\n\ngo 1.26\n\nrequire github.com/pawnkit/pawn-project v0.1.10\n")
	if err := ValidateGoMod("go.mod", valid); err != nil {
		t.Fatalf("ValidateGoMod: %v", err)
	}
	for _, content := range [][]byte{
		[]byte("module example.com/project\n\ngo 1.26\n\nreplace example.com/old => ../old\n"),
		[]byte("module example.com/project\n\ngo 1.26\n\nrequire github.com/pawnkit/pawn-project v0.0.0-20260723000000-111111111111\n"),
	} {
		if err := ValidateGoMod("go.mod", content); err == nil {
			t.Fatalf("ValidateGoMod(%q) succeeded", content)
		}
	}
}

type staticClient struct {
	response *http.Response
	err      error
}

func (client staticClient) Do(*http.Request) (*http.Response, error) {
	return client.response, client.err
}

func validSet(content []byte) Set {
	sum := sha256.Sum256(content)
	checksum := "sha256:" + hex.EncodeToString(sum[:])

	return Set{
		SchemaVersion: 1,
		ID:            "preview-2026-07-23",
		GeneratedAt:   "2026-07-23T21:00:00Z",
		Source: Source{
			Repository: "pawnkit/pawn-actions",
			Commit:     strings.Repeat("1", 40),
		},
		Targets:  []string{"linux-amd64"},
		Profiles: []string{"openmp"},
		Components: []Component{{
			Name:       "pawn",
			Repository: "pawnkit/pawnkit-cli",
			Version:    "v1.1.3",
			Commit:     strings.Repeat("2", 40),
			Artifacts: []Artifact{{
				Target:   "linux-amd64",
				URL:      "https://github.com/pawnkit/pawnkit-cli/releases/download/v1.1.3/pawn-linux-amd64.tar.gz",
				Size:     int64(len(content)),
				Checksum: checksum,
			}},
		}},
		Schemas: []Schema{{
			Name:     "pawn-project",
			Version:  1,
			URL:      "https://schemas.pawnkit.dev/pawn-project/v1/schema.json",
			Checksum: "sha256:" + strings.Repeat("3", 64),
		}},
		Evidence: Evidence{
			Workflow:    "https://github.com/pawnkit/pawn-actions/actions/runs/123456789",
			Commit:      strings.Repeat("1", 40),
			CompletedAt: "2026-07-23T21:00:00Z",
			Projects:    []string{"pawn-corpus/projects/minimal-samp-gamemode"},
			Targets:     []string{"linux-amd64"},
		},
		KnownLimits: []string{"Native plugins are experimental."},
	}
}

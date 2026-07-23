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

func TestVerifyRemote(t *testing.T) {
	t.Parallel()

	artifact := []byte("archive")
	schema := []byte(`{"type":"object"}`)
	set := validSet(artifact)
	schemaSum := sha256.Sum256(schema)
	set.Schemas[0].Checksum = "sha256:" + hex.EncodeToString(schemaSum[:])

	client := routeClient(func(request *http.Request) (*http.Response, error) {
		var content []byte
		switch request.URL.String() {
		case "https://api.github.com/repos/pawnkit/pawnkit-cli/commits/v1.1.3":
			content = []byte(`{"sha":"` + set.Components[0].Commit + `"}`)
		case set.Schemas[0].URL:
			content = schema
		case "https://api.github.com/repos/pawnkit/pawn-actions/actions/runs/123456789":
			content = []byte(`{"head_sha":"` + set.Evidence.Commit + `","status":"completed","conclusion":"success"}`)
		case set.Components[0].Artifacts[0].URL:
			content = artifact
		default:
			return response(http.StatusNotFound, nil), nil
		}
		return response(http.StatusOK, content), nil
	})

	if err := VerifyRemote(t.Context(), client, set); err != nil {
		t.Fatalf("VerifyRemote: %v", err)
	}
}

func TestVerifyRemoteRejectsWrongTagCommit(t *testing.T) {
	t.Parallel()

	set := validSet([]byte("archive"))
	client := routeClient(func(*http.Request) (*http.Response, error) {
		return response(http.StatusOK, []byte(`{"sha":"`+strings.Repeat("9", 40)+`"}`)), nil
	})
	if err := VerifyRemote(t.Context(), client, set); err == nil || !strings.Contains(err.Error(), "tag resolves") {
		t.Fatalf("VerifyRemote error = %v", err)
	}
}

func TestVerifyRemoteRejectsFailedWorkflow(t *testing.T) {
	t.Parallel()

	set := validSet([]byte("archive"))
	client := routeClient(func(request *http.Request) (*http.Response, error) {
		if strings.Contains(request.URL.Path, "/commits/") {
			return response(http.StatusOK, []byte(`{"sha":"`+set.Components[0].Commit+`"}`)), nil
		}
		if request.URL.Host == "schemas.pawnkit.dev" {
			return response(http.StatusOK, []byte{}), nil
		}
		return response(http.StatusOK, []byte(`{"head_sha":"`+set.Evidence.Commit+`","status":"completed","conclusion":"failure"}`)), nil
	})
	set.Schemas[0].Checksum = "sha256:" + strings.Repeat("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", 1)

	if err := VerifyRemote(t.Context(), client, set); err == nil || !strings.Contains(err.Error(), "conclusion failure") {
		t.Fatalf("VerifyRemote error = %v", err)
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

func TestSelectArtifact(t *testing.T) {
	t.Parallel()

	set := validSet([]byte("archive"))
	component, artifact, err := SelectArtifact(set, "pawn", "linux-amd64")
	if err != nil {
		t.Fatalf("SelectArtifact: %v", err)
	}
	if component.Version != "v1.1.3" || artifact.Target != "linux-amd64" {
		t.Fatalf("component=%+v artifact=%+v", component, artifact)
	}
	for _, test := range []struct {
		component string
		target    string
		message   string
	}{
		{component: "missing", target: "linux-amd64", message: "was not found"},
		{component: "pawn", target: "windows-amd64", message: "has no artifact"},
	} {
		if _, _, err := SelectArtifact(set, test.component, test.target); err == nil ||
			!strings.Contains(err.Error(), test.message) {
			t.Fatalf("SelectArtifact(%q, %q) error = %v", test.component, test.target, err)
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

type routeClient func(*http.Request) (*http.Response, error)

func (client routeClient) Do(request *http.Request) (*http.Response, error) {
	return client(request)
}

func response(status int, content []byte) *http.Response {
	return &http.Response{
		StatusCode:    status,
		Status:        http.StatusText(status),
		ContentLength: int64(len(content)),
		Body:          io.NopCloser(bytes.NewReader(content)),
	}
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

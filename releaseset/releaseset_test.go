package releaseset

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

func TestValidateVersionTwoModuleGraph(t *testing.T) {
	t.Parallel()

	set := validSet([]byte("archive"))
	set.SchemaVersion = 2
	set.ModuleGraph = []Module{
		{
			Repository: "pawnkit/pawnkit-cli",
			Module:     "github.com/pawnkit/pawnkit-cli",
			Version:    "v1.3.0",
			Commit:     strings.Repeat("2", 40),
			Dependencies: []Dependency{
				{
					Repository: "pawnkit/pawn-project",
					Version:    "v0.3.0",
					Kind:       "runtime",
				},
			},
		},
		{
			Repository:   "pawnkit/pawn-project",
			Module:       "github.com/pawnkit/pawn-project",
			Version:      "v0.3.0",
			Commit:       strings.Repeat("3", 40),
			Dependencies: []Dependency{},
		},
	}
	if err := set.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	set.ModuleGraph[0].Dependencies[0] = Dependency{
		Repository: "pawnkit/pawn-corpus",
		Version:    "v0.1.10",
		Kind:       "test",
	}
	if err := set.Validate(); err == nil || !strings.Contains(err.Error(), "requires provenance") {
		t.Fatalf("Validate error = %v", err)
	}
}

func TestValidateComponentModuleGraph(t *testing.T) {
	t.Parallel()

	component := Component{
		Name:       "pawn",
		Repository: "pawnkit/pawnkit-cli",
		Version:    "v1.1.3",
		Commit:     strings.Repeat("2", 40),
	}
	module := Module{
		Repository: "pawnkit/pawnkit-cli",
		Module:     "github.com/pawnkit/pawnkit-cli",
		Version:    "v1.1.3",
		Commit:     strings.Repeat("2", 40),
	}
	if err := ValidateComponentModuleGraph([]Component{component}, []Module{module}); err != nil {
		t.Fatalf("ValidateComponentModuleGraph: %v", err)
	}

	module.Commit = strings.Repeat("3", 40)
	if err := ValidateComponentModuleGraph([]Component{component}, []Module{module}); err == nil ||
		!strings.Contains(err.Error(), "differs from module graph") {
		t.Fatalf("ValidateComponentModuleGraph error = %v", err)
	}
}

func TestValidateVersionThreeSupplyChainEvidence(t *testing.T) {
	t.Parallel()

	set := validSet([]byte("archive"))
	set.SchemaVersion = 3
	set.ModuleGraph = []Module{{
		Repository: "pawnkit/pawnkit-cli",
		Module:     "github.com/pawnkit/pawnkit-cli",
		Version:    "v1.1.3",
		Commit:     strings.Repeat("2", 40),
	}}
	artifact := &set.Components[0].Artifacts[0]
	artifact.SBOM = &EvidenceAsset{
		URL:      "https://github.com/pawnkit/pawnkit-cli/releases/download/v1.1.3/pawn-linux-amd64.tar.gz.sbom.json",
		Size:     10,
		Checksum: "sha256:" + strings.Repeat("4", 64),
	}
	artifact.Provenance = &Provenance{
		Repository: "pawnkit/pawnkit-cli",
		Workflow:   "https://github.com/pawnkit/pawnkit-cli/.github/workflows/release.yml@refs/tags/v1.1.3",
		Subject:    artifact.Checksum,
	}
	if err := set.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	artifact.Provenance.Subject = "sha256:" + strings.Repeat("5", 64)
	if err := set.Validate(); err == nil || !strings.Contains(err.Error(), "mismatched provenance") {
		t.Fatalf("Validate error = %v", err)
	}
}

func TestValidateVersionThreeRequiresSupplyChainEvidence(t *testing.T) {
	t.Parallel()

	set := validSet([]byte("archive"))
	set.SchemaVersion = 3
	set.ModuleGraph = []Module{{
		Repository: "pawnkit/pawnkit-cli",
		Module:     "github.com/pawnkit/pawnkit-cli",
		Version:    "v1.1.3",
		Commit:     strings.Repeat("2", 40),
	}}
	if err := set.Validate(); err == nil || !strings.Contains(err.Error(), "requires SBOM and provenance") {
		t.Fatalf("Validate error = %v", err)
	}
}

func TestSignerWorkflow(t *testing.T) {
	t.Parallel()

	got, err := SignerWorkflow(Provenance{
		Workflow: "https://github.com/pawnkit/pawnkit-cli/.github/workflows/release.yml@refs/tags/v1.5.0",
	})
	if err != nil {
		t.Fatalf("SignerWorkflow: %v", err)
	}
	const want = "pawnkit/pawnkit-cli/.github/workflows/release.yml"
	if got != want {
		t.Fatalf("SignerWorkflow = %q, want %q", got, want)
	}
}

func TestVerifyAttestations(t *testing.T) {
	t.Parallel()

	content := []byte("archive")
	set := validVersionThreeSet(content)
	client := staticClient{response: response(http.StatusOK, content)}
	verifier := &recordingVerifier{}
	if err := VerifyAttestations(t.Context(), client, verifier, set); err != nil {
		t.Fatalf("VerifyAttestations: %v", err)
	}
	if verifier.calls != 1 {
		t.Fatalf("verifier calls = %d, want 1", verifier.calls)
	}
}

func validVersionThreeSet(content []byte) Set {
	set := validSet(content)
	set.SchemaVersion = 3
	set.ModuleGraph = []Module{{
		Repository: "pawnkit/pawnkit-cli",
		Module:     "github.com/pawnkit/pawnkit-cli",
		Version:    "v1.1.3",
		Commit:     strings.Repeat("2", 40),
	}}
	artifact := &set.Components[0].Artifacts[0]
	artifact.SBOM = &EvidenceAsset{
		URL:      "https://github.com/pawnkit/pawnkit-cli/releases/download/v1.1.3/pawn-linux-amd64.tar.gz.sbom.json",
		Size:     int64(len(content)),
		Checksum: artifact.Checksum,
	}
	artifact.Provenance = &Provenance{
		Repository: "pawnkit/pawnkit-cli",
		Workflow:   "https://github.com/pawnkit/pawnkit-cli/.github/workflows/release.yml@refs/tags/v1.1.3",
		Subject:    artifact.Checksum,
	}
	return set
}

type recordingVerifier struct {
	calls int
}

func (verifier *recordingVerifier) Verify(_ context.Context, path string, _ Component, _ Artifact) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if string(content) != "archive" {
		return fmt.Errorf("artifact content = %q", content)
	}
	verifier.calls++
	return nil
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

func TestVerifyRemoteChecksVersionTwoModules(t *testing.T) {
	t.Parallel()

	set := validSet([]byte("archive"))
	set.SchemaVersion = 2
	set.ModuleGraph = []Module{
		{
			Repository:   "pawnkit/pawn-project",
			Module:       "github.com/pawnkit/pawn-project",
			Version:      "v0.3.0",
			Commit:       strings.Repeat("3", 40),
			Dependencies: []Dependency{},
		},
	}
	client := routeClient(func(request *http.Request) (*http.Response, error) {
		if strings.Contains(request.URL.Path, "/pawnkit/pawn-project/") {
			return response(http.StatusNotFound, nil), nil
		}
		if strings.Contains(request.URL.Path, "/commits/") {
			return response(http.StatusOK, []byte(`{"sha":"`+set.Components[0].Commit+`"}`)), nil
		}
		return response(http.StatusOK, nil), nil
	})
	if err := VerifyRemote(t.Context(), client, set); err == nil ||
		!strings.Contains(err.Error(), `module "pawnkit/pawn-project"`) {
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

func TestNewRequestAuthenticatesOnlyGitHubAPI(t *testing.T) {
	t.Setenv("GH_TOKEN", "test-token")

	request, err := newRequest(t.Context(), "https://api.github.com/repos/pawnkit/pawn-actions")
	if err != nil {
		t.Fatal(err)
	}
	if got := request.Header.Get("Authorization"); got != "Bearer test-token" {
		t.Fatalf("Authorization = %q", got)
	}

	request, err = newRequest(t.Context(), "https://github.com/pawnkit/pawn-actions/releases/download/v1/tool.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if got := request.Header.Get("Authorization"); got != "" {
		t.Fatalf("release Authorization = %q", got)
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

func TestValidateGoModuleGraph(t *testing.T) {
	t.Parallel()

	modules := []GoModule{
		{
			Repository: "pawn-analysis",
			Dependencies: []GoDependency{
				{Repository: "pawn-parser", Version: "v0.1.0"},
			},
		},
		{Repository: "pawn-parser"},
	}
	if err := ValidateGoModuleGraph(modules); err != nil {
		t.Fatalf("ValidateGoModuleGraph: %v", err)
	}

	modules[1].Dependencies = []GoDependency{
		{Repository: "pawn-analysis", Version: "v0.1.0"},
	}
	if err := ValidateGoModuleGraph(modules); err == nil ||
		!strings.Contains(err.Error(), "reversed dependency") {
		t.Fatalf("ValidateGoModuleGraph error = %v", err)
	}
}

func TestValidateGoModuleGraphRejectsSameLayerCycle(t *testing.T) {
	t.Parallel()

	modules := []GoModule{
		{
			Repository: "pawnlint",
			Dependencies: []GoDependency{
				{Repository: "pawnfmt", Version: "v1.0.0"},
			},
		},
		{
			Repository: "pawnfmt",
			Dependencies: []GoDependency{
				{Repository: "pawnlint", Version: "v1.0.0"},
			},
		},
	}
	if err := ValidateGoModuleGraph(modules); err == nil ||
		!strings.Contains(err.Error(), "dependency cycle") {
		t.Fatalf("ValidateGoModuleGraph error = %v", err)
	}
}

func TestValidateGoModuleGraphAllowsSeveralReleasedVersions(t *testing.T) {
	t.Parallel()

	modules := []GoModule{
		{Repository: "pawn-project", Version: "v0.1.9"},
		{Repository: "pawn-project", Version: "v0.3.0"},
	}
	if err := ValidateGoModuleGraph(modules); err != nil {
		t.Fatalf("ValidateGoModuleGraph: %v", err)
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

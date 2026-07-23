// Package releaseset validates PawnKit tested release sets.
package releaseset

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
)

const (
	maxDocumentSize = 4 << 20
	maxArtifactSize = 2 << 30
)

var (
	idPattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	profilePattern  = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	versionPattern  = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)
	commitPattern   = regexp.MustCompile(`^[0-9a-f]{40}$`)
	repoPattern     = regexp.MustCompile(`^pawnkit/[a-z0-9._-]+$`)
	workflowPattern = regexp.MustCompile(`^/pawnkit/[a-z0-9._-]+/actions/runs/[0-9]+$`)
)

var supportedTargets = []string{
	"linux-amd64", "linux-arm64", "linux-386",
	"windows-amd64", "windows-arm64", "windows-386",
	"darwin-amd64", "darwin-arm64", "darwin-386",
}

type Set struct {
	SchemaVersion int         `json:"schemaVersion"`
	ID            string      `json:"id"`
	GeneratedAt   string      `json:"generatedAt"`
	Source        Source      `json:"source"`
	Targets       []string    `json:"targets"`
	Profiles      []string    `json:"profiles"`
	Components    []Component `json:"components"`
	Schemas       []Schema    `json:"schemas"`
	Evidence      Evidence    `json:"evidence"`
	KnownLimits   []string    `json:"knownLimits"`
}

type Source struct {
	Repository string `json:"repository"`
	Commit     string `json:"commit"`
}

type Component struct {
	Name       string     `json:"name"`
	Repository string     `json:"repository"`
	Version    string     `json:"version"`
	Commit     string     `json:"commit"`
	Artifacts  []Artifact `json:"artifacts,omitempty"`
}

type Artifact struct {
	Target   string `json:"target"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
}

type Schema struct {
	Name     string `json:"name"`
	Version  int    `json:"version"`
	URL      string `json:"url"`
	Checksum string `json:"checksum"`
}

type Evidence struct {
	Workflow    string   `json:"workflow"`
	Commit      string   `json:"commit"`
	CompletedAt string   `json:"completedAt"`
	Projects    []string `json:"projects"`
	Targets     []string `json:"targets"`
}

func Decode(reader io.Reader) (Set, error) {
	content, err := io.ReadAll(io.LimitReader(reader, maxDocumentSize+1))
	if err != nil {
		return Set{}, fmt.Errorf("release set: read: %w", err)
	}
	if len(content) > maxDocumentSize {
		return Set{}, errors.New("release set: document is too large")
	}

	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()

	var set Set
	if err := decoder.Decode(&set); err != nil {
		return Set{}, fmt.Errorf("release set: decode: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return Set{}, err
	}
	if err := set.Validate(); err != nil {
		return Set{}, err
	}
	return set, nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New("release set: multiple JSON values")
	}
	return fmt.Errorf("release set: trailing data: %w", err)
}

func (set Set) Validate() error {
	if set.SchemaVersion != 1 {
		return fmt.Errorf("release set: unsupported schema version %d", set.SchemaVersion)
	}
	if !idPattern.MatchString(set.ID) {
		return errors.New("release set: invalid id")
	}
	generatedAt, err := time.Parse(time.RFC3339, set.GeneratedAt)
	if err != nil {
		return errors.New("release set: invalid generatedAt")
	}
	if err := validateSource(set.Source); err != nil {
		return err
	}
	if err := validateUnique("target", set.Targets, func(value string) bool {
		return slices.Contains(supportedTargets, value)
	}); err != nil {
		return err
	}
	if err := validateUnique("profile", set.Profiles, profilePattern.MatchString); err != nil {
		return err
	}
	if len(set.Components) == 0 || len(set.Components) > 128 {
		return errors.New("release set: components must contain 1 to 128 entries")
	}
	if err := validateComponents(set.Components, set.Targets); err != nil {
		return err
	}
	if err := validateSchemas(set.Schemas); err != nil {
		return err
	}
	completedAt, err := validateEvidence(set.Evidence, set.Targets)
	if err != nil {
		return err
	}
	if set.Evidence.Commit != set.Source.Commit {
		return errors.New("release set: evidence commit does not match source commit")
	}
	if generatedAt.Before(completedAt) {
		return errors.New("release set: generatedAt predates test completion")
	}
	if set.KnownLimits == nil {
		return errors.New("release set: knownLimits is required")
	}
	if len(set.KnownLimits) > 128 {
		return errors.New("release set: too many known limits")
	}
	for _, limit := range set.KnownLimits {
		if limit == "" || len(limit) > 1024 {
			return errors.New("release set: invalid known limit")
		}
	}
	return nil
}

func validateSource(source Source) error {
	if !repoPattern.MatchString(source.Repository) || !commitPattern.MatchString(source.Commit) {
		return errors.New("release set: invalid source")
	}
	return nil
}

func validateUnique(label string, values []string, valid func(string) bool) error {
	if len(values) == 0 || len(values) > 64 {
		return fmt.Errorf("release set: %ss must contain 1 to 64 entries", label)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !valid(value) {
			return fmt.Errorf("release set: invalid %s %q", label, value)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("release set: duplicate %s %q", label, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateComponents(components []Component, targets []string) error {
	seen := make(map[string]struct{}, len(components))
	for _, component := range components {
		if !idPattern.MatchString(component.Name) ||
			!repoPattern.MatchString(component.Repository) ||
			!versionPattern.MatchString(component.Version) ||
			!commitPattern.MatchString(component.Commit) {
			return fmt.Errorf("release set: invalid component %q", component.Name)
		}
		if _, exists := seen[component.Name]; exists {
			return fmt.Errorf("release set: duplicate component %q", component.Name)
		}
		seen[component.Name] = struct{}{}
		if len(component.Artifacts) > 16 {
			return fmt.Errorf("release set: component %q has too many artifacts", component.Name)
		}
		artifactTargets := make(map[string]struct{}, len(component.Artifacts))
		for _, artifact := range component.Artifacts {
			if !slices.Contains(targets, artifact.Target) {
				return fmt.Errorf("release set: component %q has untested target %q", component.Name, artifact.Target)
			}
			if _, exists := artifactTargets[artifact.Target]; exists {
				return fmt.Errorf("release set: component %q repeats target %q", component.Name, artifact.Target)
			}
			artifactTargets[artifact.Target] = struct{}{}
			if err := validateArtifact(component, artifact); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateArtifact(component Component, artifact Artifact) error {
	if artifact.Size < 1 || artifact.Size > maxArtifactSize {
		return fmt.Errorf("release set: component %q has invalid artifact size", component.Name)
	}
	if _, err := checksumBytes(artifact.Checksum); err != nil {
		return fmt.Errorf("release set: component %q: %w", component.Name, err)
	}
	parsed, err := url.Parse(artifact.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "github.com" {
		return fmt.Errorf("release set: component %q has invalid artifact URL", component.Name)
	}
	wantPrefix := "/" + component.Repository + "/releases/download/" + component.Version + "/"
	if !strings.HasPrefix(parsed.EscapedPath(), wantPrefix) {
		return fmt.Errorf("release set: component %q artifact URL does not match its release", component.Name)
	}
	return nil
}

func validateSchemas(schemas []Schema) error {
	if len(schemas) == 0 || len(schemas) > 64 {
		return errors.New("release set: schemas must contain 1 to 64 entries")
	}
	seen := make(map[string]struct{}, len(schemas))
	for _, schema := range schemas {
		if !idPattern.MatchString(schema.Name) || schema.Version < 1 {
			return fmt.Errorf("release set: invalid schema %q", schema.Name)
		}
		if _, exists := seen[schema.Name]; exists {
			return fmt.Errorf("release set: duplicate schema %q", schema.Name)
		}
		seen[schema.Name] = struct{}{}
		if _, err := checksumBytes(schema.Checksum); err != nil {
			return fmt.Errorf("release set: schema %q: %w", schema.Name, err)
		}
		want := fmt.Sprintf("https://schemas.pawnkit.dev/%s/v%d/schema.json", schema.Name, schema.Version)
		if schema.URL != want {
			return fmt.Errorf("release set: schema %q has unexpected URL", schema.Name)
		}
	}
	return nil
}

func validateEvidence(evidence Evidence, targets []string) (time.Time, error) {
	parsed, err := url.Parse(evidence.Workflow)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "github.com" ||
		!workflowPattern.MatchString(parsed.Path) {
		return time.Time{}, errors.New("release set: invalid evidence workflow")
	}
	if !commitPattern.MatchString(evidence.Commit) {
		return time.Time{}, errors.New("release set: invalid evidence commit")
	}
	completedAt, err := time.Parse(time.RFC3339, evidence.CompletedAt)
	if err != nil {
		return time.Time{}, errors.New("release set: invalid evidence completedAt")
	}
	if err := validateUnique("evidence project", evidence.Projects, func(value string) bool {
		return value != "" && len(value) <= 512
	}); err != nil {
		return time.Time{}, err
	}
	if err := validateUnique("evidence target", evidence.Targets, func(value string) bool {
		return slices.Contains(targets, value)
	}); err != nil {
		return time.Time{}, err
	}
	if !sameStrings(targets, evidence.Targets) {
		return time.Time{}, errors.New("release set: declared targets do not match tested targets")
	}
	return completedAt, nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for _, value := range left {
		if !slices.Contains(right, value) {
			return false
		}
	}
	return true
}

func checksumBytes(value string) ([]byte, error) {
	raw, ok := strings.CutPrefix(value, "sha256:")
	if !ok {
		return nil, errors.New("checksum must use sha256")
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil || len(decoded) != sha256.Size {
		return nil, errors.New("invalid sha256 checksum")
	}
	return decoded, nil
}

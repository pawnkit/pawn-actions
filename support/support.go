// Package support validates PawnKit repository support records.
package support

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"time"
)

const maxDocumentSize = 1 << 20

var (
	repositoryPattern = regexp.MustCompile(`^pawnkit/[a-z0-9][a-z0-9._-]*$`)
	profilePattern    = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	namePattern       = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	versionPattern    = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
)

var (
	maturities = []string{"stable", "preview", "experimental", "infrastructure", "deprecated"}
	targets    = []string{
		"linux-amd64", "linux-arm64", "linux-386",
		"windows-amd64", "windows-arm64", "windows-386",
		"darwin-amd64", "darwin-arm64", "darwin-386",
	}
	compilerNames = []string{"pawncc", "openmp-pawncc"}
)

type Record struct {
	SchemaVersion int         `json:"schemaVersion"`
	Repository    string      `json:"repository"`
	Maturity      string      `json:"maturity"`
	Platforms     []string    `json:"platforms"`
	Profiles      []string    `json:"profiles"`
	Compilers     []Compiler  `json:"compilers"`
	Contracts     []Contract  `json:"contracts"`
	Limitations   []string    `json:"limitations"`
	Deprecated    *Deprecated `json:"deprecated,omitempty"`
}

type Compiler struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Contract struct {
	Name  string `json:"name"`
	Major int    `json:"major"`
}

type Deprecated struct {
	Replacement      string `json:"replacement,omitempty"`
	EndOfSupport     string `json:"endOfSupport,omitempty"`
	RemovalMilestone string `json:"removalMilestone,omitempty"`
}

func Decode(reader io.Reader) (Record, error) {
	content, err := io.ReadAll(io.LimitReader(reader, maxDocumentSize+1))
	if err != nil {
		return Record{}, fmt.Errorf("support: read: %w", err)
	}
	if len(content) > maxDocumentSize {
		return Record{}, errors.New("support: document exceeds 1 MiB")
	}
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	var record Record
	if err := decoder.Decode(&record); err != nil {
		return Record{}, fmt.Errorf("support: decode: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Record{}, errors.New("support: multiple JSON values")
		}
		return Record{}, fmt.Errorf("support: trailing data: %w", err)
	}
	if err := record.Validate("", nil); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (record Record) Validate(repository string, testedTargets []string) error {
	if record.SchemaVersion != 1 {
		return fmt.Errorf("support: unsupported schema version %d", record.SchemaVersion)
	}
	if !repositoryPattern.MatchString(record.Repository) {
		return errors.New("support: invalid repository")
	}
	if repository != "" && record.Repository != repository {
		return fmt.Errorf("support: repository is %q, want %q", record.Repository, repository)
	}
	if !slices.Contains(maturities, record.Maturity) {
		return fmt.Errorf("support: invalid maturity %q", record.Maturity)
	}
	if err := validateStrings("platform", record.Platforms, func(value string) bool {
		return slices.Contains(targets, value)
	}); err != nil {
		return err
	}
	if err := validateStrings("profile", record.Profiles, profilePattern.MatchString); err != nil {
		return err
	}
	if err := validateCompilers(record.Compilers); err != nil {
		return err
	}
	if err := validateContracts(record.Contracts); err != nil {
		return err
	}
	if len(record.Limitations) > 128 {
		return errors.New("support: too many limitations")
	}
	for _, limitation := range record.Limitations {
		if limitation == "" || len(limitation) > 512 {
			return errors.New("support: invalid limitation")
		}
	}
	if err := validateDeprecation(record); err != nil {
		return err
	}
	if len(testedTargets) != 0 {
		for _, platform := range record.Platforms {
			if !slices.Contains(testedTargets, platform) {
				return fmt.Errorf("support: platform %q is not covered by CI", platform)
			}
		}
	}
	return nil
}

func validateStrings(label string, values []string, valid func(string) bool) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !valid(value) {
			return fmt.Errorf("support: invalid %s %q", label, value)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("support: duplicate %s %q", label, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateCompilers(compilers []Compiler) error {
	seen := make(map[string]struct{}, len(compilers))
	for _, compiler := range compilers {
		key := compiler.Name + "@" + compiler.Version
		if !slices.Contains(compilerNames, compiler.Name) ||
			!versionPattern.MatchString(compiler.Version) {
			return fmt.Errorf("support: invalid compiler %q", key)
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("support: duplicate compiler %q", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validateContracts(contracts []Contract) error {
	seen := make(map[string]struct{}, len(contracts))
	for _, contract := range contracts {
		if !namePattern.MatchString(contract.Name) || contract.Major < 1 {
			return fmt.Errorf("support: invalid contract %q", contract.Name)
		}
		if _, exists := seen[contract.Name]; exists {
			return fmt.Errorf("support: duplicate contract %q", contract.Name)
		}
		seen[contract.Name] = struct{}{}
	}
	return nil
}

func validateDeprecation(record Record) error {
	if record.Maturity != "deprecated" {
		if record.Deprecated != nil {
			return errors.New("support: deprecation data requires deprecated maturity")
		}
		return nil
	}
	if record.Deprecated == nil {
		return errors.New("support: deprecated maturity requires deprecation data")
	}
	if record.Deprecated.EndOfSupport == "" && record.Deprecated.RemovalMilestone == "" {
		return errors.New("support: deprecation requires an end date or removal milestone")
	}
	if record.Deprecated.EndOfSupport != "" {
		if _, err := time.Parse(time.DateOnly, record.Deprecated.EndOfSupport); err != nil {
			return errors.New("support: invalid end-of-support date")
		}
	}
	return nil
}

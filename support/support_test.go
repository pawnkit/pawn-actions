package support_test

import (
	"strings"
	"testing"

	"github.com/pawnkit/pawn-actions/support"
)

func TestDecodeAndValidate(t *testing.T) {
	t.Parallel()

	record, err := support.Decode(strings.NewReader(`{
		"schemaVersion": 1,
		"repository": "pawnkit/pawnlint",
		"maturity": "preview",
		"platforms": ["linux-amd64"],
		"profiles": ["openmp"],
		"compilers": [{"name":"pawncc","version":"3.10.10"}],
		"contracts": [{"name":"pawn-diagnostic","major":2}],
		"limitations": []
	}`))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if err := record.Validate("pawnkit/pawnlint", []string{"linux-amd64"}); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidateRejectsUnsupportedPlatform(t *testing.T) {
	t.Parallel()

	record := support.Record{
		SchemaVersion: 1,
		Repository:    "pawnkit/pawnlint",
		Maturity:      "preview",
		Platforms:     []string{"windows-amd64"},
		Profiles:      []string{},
		Compilers:     []support.Compiler{},
		Contracts:     []support.Contract{},
		Limitations:   []string{},
	}
	if err := record.Validate("pawnkit/pawnlint", []string{"linux-amd64"}); err == nil ||
		!strings.Contains(err.Error(), "not covered by CI") {
		t.Fatalf("Validate error = %v", err)
	}
}

func TestValidateRejectsDuplicateContract(t *testing.T) {
	t.Parallel()

	record := support.Record{
		SchemaVersion: 1,
		Repository:    "pawnkit/pawnlint",
		Maturity:      "preview",
		Platforms:     []string{},
		Profiles:      []string{},
		Compilers:     []support.Compiler{},
		Contracts: []support.Contract{
			{Name: "pawn-project", Major: 1},
			{Name: "pawn-project", Major: 2},
		},
		Limitations: []string{},
	}
	if err := record.Validate("", nil); err == nil ||
		!strings.Contains(err.Error(), "duplicate contract") {
		t.Fatalf("Validate error = %v", err)
	}
}

func TestValidateDeprecation(t *testing.T) {
	t.Parallel()

	record := support.Record{
		SchemaVersion: 1,
		Repository:    "pawnkit/old-tool",
		Maturity:      "deprecated",
		Platforms:     []string{},
		Profiles:      []string{},
		Compilers:     []support.Compiler{},
		Contracts:     []support.Contract{},
		Limitations:   []string{},
		Deprecated:    &support.Deprecated{EndOfSupport: "not-a-date"},
	}
	if err := record.Validate("", nil); err == nil ||
		!strings.Contains(err.Error(), "end-of-support date") {
		t.Fatalf("Validate error = %v", err)
	}
}

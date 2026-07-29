package main

import (
	"slices"
	"testing"

	"github.com/pawnkit/pawn-actions/releaseset"
)

func TestGHVerificationArgumentsUseOneSignerConstraint(t *testing.T) {
	t.Parallel()

	arguments, err := ghVerificationArguments("pawn.tar.gz", releaseset.Component{
		Repository: "pawnkit/pawn",
		Version:    "v1.2.3",
		Commit:     "0123456789abcdef",
	}, releaseset.Artifact{
		Provenance: &releaseset.Provenance{
			Workflow: "https://github.com/pawnkit/pawn/.github/workflows/release.yml@refs/tags/v1.2.3",
		},
	})
	if err != nil {
		t.Fatalf("ghVerificationArguments: %v", err)
	}
	if slices.Contains(arguments, "--signer-repo") {
		t.Fatal("arguments contain mutually exclusive --signer-repo")
	}
	if !slices.Contains(arguments, "--signer-workflow") {
		t.Fatal("arguments do not constrain the signer workflow")
	}
}

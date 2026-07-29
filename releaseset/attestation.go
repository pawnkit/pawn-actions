package releaseset

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type AttestationVerifier interface {
	Verify(context.Context, string, Component, Artifact) error
}

func VerifyAttestations(ctx context.Context, client HTTPClient, verifier AttestationVerifier, set Set) error {
	if client == nil {
		return fmt.Errorf("release set: HTTP client is required")
	}
	if verifier == nil {
		return fmt.Errorf("release set: attestation verifier is required")
	}
	if err := set.Validate(); err != nil {
		return err
	}
	if set.SchemaVersion < 3 {
		return fmt.Errorf("release set: attestation verification requires schema version 3")
	}

	directory, err := os.MkdirTemp("", "pawn-release-attestations-")
	if err != nil {
		return fmt.Errorf("release set: create attestation directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(directory)
	}()

	for _, component := range set.Components {
		for index, artifact := range component.Artifacts {
			path := filepath.Join(directory, fmt.Sprintf("%s-%d", component.Name, index))
			if err := downloadAsset(ctx, client, component.Name, artifact.URL, artifact.Size, artifact.Checksum, path); err != nil {
				return err
			}
			if err := verifier.Verify(ctx, path, component, artifact); err != nil {
				return fmt.Errorf("release set: component %q target %q: verify attestation: %w", component.Name, artifact.Target, err)
			}
		}
	}
	return nil
}

func SignerWorkflow(provenance Provenance) (string, error) {
	const prefix = "https://github.com/"
	workflow, _, ok := strings.Cut(strings.TrimPrefix(provenance.Workflow, prefix), "@")
	if !strings.HasPrefix(provenance.Workflow, prefix) || !ok || workflow == "" {
		return "", fmt.Errorf("invalid provenance workflow")
	}
	return workflow, nil
}

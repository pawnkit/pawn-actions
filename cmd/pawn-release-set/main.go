// Command pawn-release-set validates a tested release set.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pawnkit/pawn-actions/releaseset"
)

type paths []string

func (values *paths) String() string {
	return fmt.Sprint([]string(*values))
}

func (values *paths) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func main() {
	var modules paths
	verify := flag.Bool("verify-artifacts", false, "download and verify release artifacts")
	verifyRemote := flag.Bool("verify-remote", false, "verify tags, schemas, workflow evidence, and artifacts")
	component := flag.String("component", "", "select an artifact from this component")
	target := flag.String("target", "", "select an artifact for this target")
	githubOutput := flag.String("github-output", "", "append the selected artifact to a GitHub output file")
	flag.Var(&modules, "go-mod", "check a go.mod file; repeat for more files")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: pawn-release-set [options] <set.json>")
		os.Exit(2)
	}
	options := runOptions{
		verifyArtifacts: *verify,
		verifyRemote:    *verifyRemote,
		component:       *component,
		target:          *target,
		githubOutput:    *githubOutput,
	}
	if err := run(flag.Arg(0), modules, options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type runOptions struct {
	verifyArtifacts bool
	verifyRemote    bool
	component       string
	target          string
	githubOutput    string
}

func run(path string, modules []string, options runOptions) error {
	file, err := os.Open(path) //nolint:gosec // The path is an explicit CLI argument.
	if err != nil {
		return fmt.Errorf("open release set: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	set, err := releaseset.Decode(file)
	if err != nil {
		return err
	}
	for _, modulePath := range modules {
		content, readErr := os.ReadFile(modulePath) //nolint:gosec // The path is an explicit CLI argument.
		if readErr != nil {
			return fmt.Errorf("read %s: %w", modulePath, readErr)
		}
		if err := releaseset.ValidateGoMod(modulePath, content); err != nil {
			return err
		}
	}
	if options.verifyRemote {
		client := &http.Client{Timeout: 5 * time.Minute}
		if err := releaseset.VerifyRemote(context.Background(), client, set); err != nil {
			return err
		}
	} else if options.verifyArtifacts {
		client := &http.Client{Timeout: 5 * time.Minute}
		if err := releaseset.VerifyArtifacts(context.Background(), client, set); err != nil {
			return err
		}
	}
	if err := selectArtifact(set, options); err != nil {
		return err
	}

	fmt.Printf("ok: %s (%d components)\n", set.ID, len(set.Components))
	return nil
}

func selectArtifact(set releaseset.Set, options runOptions) error {
	selecting := options.component != "" || options.target != "" || options.githubOutput != ""
	if !selecting {
		return nil
	}
	if options.component == "" || options.target == "" {
		return fmt.Errorf("select an artifact with both -component and -target")
	}
	component, artifact, err := releaseset.SelectArtifact(set, options.component, options.target)
	if err != nil {
		return err
	}
	if options.githubOutput == "" {
		fmt.Printf("%s %s %s\n", component.Version, artifact.URL, artifact.Checksum)
		return nil
	}
	output, err := os.OpenFile(options.githubOutput, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec
	if err != nil {
		return fmt.Errorf("open GitHub output: %w", err)
	}
	defer func() {
		_ = output.Close()
	}()
	values := []string{
		"component=" + component.Name,
		"repository=" + component.Repository,
		"version=" + component.Version,
		"commit=" + component.Commit,
		"target=" + artifact.Target,
		"url=" + artifact.URL,
		"size=" + strconv.FormatInt(artifact.Size, 10),
		"sha256=" + strings.TrimPrefix(artifact.Checksum, "sha256:"),
	}
	if _, err := fmt.Fprintln(output, strings.Join(values, "\n")); err != nil {
		return fmt.Errorf("write GitHub output: %w", err)
	}
	return nil
}

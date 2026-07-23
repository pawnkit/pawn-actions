// Command pawn-release-set validates a tested release set.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
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
	flag.Var(&modules, "go-mod", "check a go.mod file; repeat for more files")
	flag.Parse()

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: pawn-release-set [options] <set.json>")
		os.Exit(2)
	}
	if err := run(flag.Arg(0), modules, *verify); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path string, modules []string, verify bool) error {
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
	if verify {
		client := &http.Client{Timeout: 5 * time.Minute}
		if err := releaseset.VerifyArtifacts(context.Background(), client, set); err != nil {
			return err
		}
	}

	fmt.Printf("ok: %s (%d components)\n", set.ID, len(set.Components))
	return nil
}

// Command pawn-support-check validates a repository support record.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/pawnkit/pawn-actions/support"
)

func main() {
	repository := flag.String("repository", "", "expected pawnkit/name repository")
	targets := flag.String("targets", "", "comma-separated targets covered by CI")
	flag.Parse()
	if flag.NArg() != 1 || *repository == "" {
		fmt.Fprintln(os.Stderr, "usage: pawn-support-check -repository pawnkit/name [-targets list] <support.json>")
		os.Exit(2)
	}
	if err := run(flag.Arg(0), *repository, splitTargets(*targets)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path string, repository string, targets []string) error {
	file, err := os.Open(path) //nolint:gosec // The path is an explicit argument.
	if err != nil {
		return fmt.Errorf("open support record: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()
	record, err := support.Decode(file)
	if err != nil {
		return err
	}
	if err := record.Validate(repository, targets); err != nil {
		return err
	}
	fmt.Printf("ok: %s (%s)\n", record.Repository, record.Maturity)
	return nil
}

func splitTargets(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

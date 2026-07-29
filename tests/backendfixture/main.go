// Command backendfixture exercises the PawnKit run protocol in CI.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

type request struct {
	Kind          string `json:"kind"`
	SchemaVersion int    `json:"schemaVersion"`
	Operation     string `json:"operation"`
	Output        string `json:"output"`
}

func main() {
	if len(os.Args) < 2 {
		fail(errors.New("command is required"))
	}

	switch os.Args[1] {
	case "capabilities":
		writeJSON(os.Stdout, map[string]any{
			"kind":            "capabilities",
			"protocolVersion": 1,
			"name":            "pawn-actions-run-fixture",
			"version":         "1.0.0",
			"operations":      []string{"run"},
			"profiles":        []string{"samp-037", "openmp"},
		})
	case "execute":
		execute(os.Args[2:])
	default:
		fail(fmt.Errorf("unknown command %q", os.Args[1]))
	}
}

func execute(args []string) {
	flags := flag.NewFlagSet("execute", flag.ContinueOnError)
	input := flags.String("input", "", "request file")
	output := flags.String("output", "", "result file")
	if err := flags.Parse(args); err != nil {
		fail(err)
	}

	body, err := os.ReadFile(*input)
	if err != nil {
		fail(err)
	}

	var value request
	if err := json.Unmarshal(body, &value); err != nil {
		fail(err)
	}
	if value.Kind != "request" || value.SchemaVersion != 2 || value.Operation != "run" {
		fail(errors.New("invalid run request"))
	}
	if _, err := os.Stat(value.Output); err != nil {
		fail(fmt.Errorf("run artifact: %w", err))
	}

	file, err := os.Create(*output)
	if err != nil {
		fail(err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fail(err)
		}
	}()

	writeJSON(file, map[string]any{
		"kind":          "result",
		"schemaVersion": 2,
		"status":        "passed",
		"backend": map[string]any{
			"name":    "pawn-actions-run-fixture",
			"version": "1.0.0",
		},
		"artifacts":   []any{},
		"diagnostics": []any{},
		"process": map[string]any{
			"exitCode": 0,
			"stdout":   "run protocol passed\n",
		},
	})
}

func writeJSON(file *os.File, value any) {
	if err := json.NewEncoder(file).Encode(value); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

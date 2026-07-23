package main

import (
	"strings"
	"testing"
)

func TestValidateOptions(t *testing.T) {
	t.Parallel()

	valid := options{
		binary:      "pawnlint",
		version:     "v1.1.4",
		archiveURL:  "https://github.com/pawnkit/pawnlint/releases/download/v1.1.4/pawnlint.zip",
		checksum:    strings.Repeat("a", 64),
		destination: t.TempDir(),
	}
	if _, err := validateOptions(valid); err != nil {
		t.Fatalf("validateOptions: %v", err)
	}

	tests := []options{
		func() options { value := valid; value.binary = "../pawnlint"; return value }(),
		func() options { value := valid; value.version = "latest"; return value }(),
		func() options { value := valid; value.archiveURL = "http://example.com/tool.zip"; return value }(),
		func() options { value := valid; value.checksum = "bad"; return value }(),
		func() options { value := valid; value.destination = "."; return value }(),
	}
	for _, test := range tests {
		if _, err := validateOptions(test); err == nil {
			t.Fatalf("validateOptions(%+v) succeeded", test)
		}
	}
}

func TestLimitedBuffer(t *testing.T) {
	t.Parallel()

	var output limitedBuffer
	if _, err := output.Write(make([]byte, maxVersionOutput)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := output.Write([]byte{0}); err == nil {
		t.Fatal("oversized write succeeded")
	}
}

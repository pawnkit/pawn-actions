package toolinstall

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"testing"
)

func TestExtractTarGz(t *testing.T) {
	t.Parallel()

	content := tarArchive(t, map[string]string{
		"pawntest":             "binary",
		"include/pawntest.inc": "include",
		"README.md":            "ignored",
	})
	files, err := Extract("pawntest.tar.gz", content, "pawntest")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("files = %+v", files)
	}
}

func TestExtractZip(t *testing.T) {
	t.Parallel()

	var content bytes.Buffer
	writer := zip.NewWriter(&content)
	for name, value := range map[string]string{"pawnlint.exe": "binary", "README.md": "ignored"} {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(value)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	files, err := Extract("pawnlint.zip", content.Bytes(), "pawnlint")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(files) != 1 || files[0].Name != "pawnlint.exe" {
		t.Fatalf("files = %+v", files)
	}
}

func TestExtractRejectsUnsafeArchive(t *testing.T) {
	t.Parallel()

	for _, files := range []map[string]string{
		{"../pawnlint": "binary"},
		{"include/../pawnlint.inc": "include", "pawnlint": "binary"},
		{"README.md": "missing binary"},
	} {
		if _, err := Extract("pawnlint.tar.gz", tarArchive(t, files), "pawnlint"); err == nil {
			t.Fatalf("Extract(%v) succeeded", files)
		}
	}
}

func tarArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var content bytes.Buffer
	compressed := gzip.NewWriter(&content)
	writer := tar.NewWriter(compressed)
	for name, value := range files {
		data := []byte(value)
		if err := writer.WriteHeader(&tar.Header{
			Name: name,
			Mode: 0o755,
			Size: int64(len(data)),
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	return content.Bytes()
}

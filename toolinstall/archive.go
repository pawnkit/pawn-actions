// Package toolinstall reads PawnKit release archives.
package toolinstall

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

const MaxArchiveSize = 200 << 20

type File struct {
	Name string
	Mode uint32
	Data []byte
}

func Extract(name string, content []byte, binary string) ([]File, error) {
	switch {
	case strings.HasSuffix(name, ".tar.gz"):
		return extractTarGz(content, binary)
	case strings.HasSuffix(name, ".zip"):
		return extractZip(content, binary)
	default:
		return nil, errors.New("tool archive must be .tar.gz or .zip")
	}
}

func extractTarGz(content []byte, binary string) ([]File, error) {
	compressed, err := gzip.NewReader(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("read gzip: %w", err)
	}
	defer func() {
		_ = compressed.Close()
	}()

	reader := tar.NewReader(compressed)
	files := make([]File, 0, 8)
	var total int64
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		if header.Typeflag != tar.TypeReg {
			return nil, fmt.Errorf("archive entry %q is not a regular file", header.Name)
		}
		selected, clean, err := selectEntry(header.Name, binary)
		if err != nil {
			return nil, err
		}
		if !selected {
			continue
		}
		total += header.Size
		if header.Size < 0 || total > MaxArchiveSize {
			return nil, errors.New("extracted tool files are too large")
		}
		data, err := io.ReadAll(io.LimitReader(reader, header.Size+1))
		if err != nil {
			return nil, fmt.Errorf("read archive entry %q: %w", header.Name, err)
		}
		if int64(len(data)) != header.Size {
			return nil, fmt.Errorf("archive entry %q has the wrong size", header.Name)
		}
		files = append(files, File{Name: clean, Mode: uint32(header.Mode), Data: data}) //nolint:gosec
	}
	return validateFiles(files, binary)
}

func extractZip(content []byte, binary string) ([]File, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("read zip: %w", err)
	}
	files := make([]File, 0, 8)
	var total uint64
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		if !entry.Mode().IsRegular() {
			return nil, fmt.Errorf("archive entry %q is not a regular file", entry.Name)
		}
		selected, clean, err := selectEntry(entry.Name, binary)
		if err != nil {
			return nil, err
		}
		if !selected {
			continue
		}
		total += entry.UncompressedSize64
		if total > MaxArchiveSize {
			return nil, errors.New("extracted tool files are too large")
		}
		handle, err := entry.Open()
		if err != nil {
			return nil, fmt.Errorf("open archive entry %q: %w", entry.Name, err)
		}
		data, readErr := io.ReadAll(io.LimitReader(handle, int64(entry.UncompressedSize64)+1)) //nolint:gosec
		closeErr := handle.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read archive entry %q: %w", entry.Name, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close archive entry %q: %w", entry.Name, closeErr)
		}
		if uint64(len(data)) != entry.UncompressedSize64 {
			return nil, fmt.Errorf("archive entry %q has the wrong size", entry.Name)
		}
		files = append(files, File{Name: clean, Mode: uint32(entry.Mode().Perm()), Data: data})
	}
	return validateFiles(files, binary)
}

func selectEntry(name, binary string) (bool, string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || path.IsAbs(clean) || clean != name {
		return false, "", fmt.Errorf("archive entry %q has an unsafe path", name)
	}
	base := path.Base(clean)
	if base == binary || base == binary+".exe" {
		return true, base, nil
	}
	if strings.HasPrefix(clean, "include/") && strings.HasSuffix(clean, ".inc") {
		return true, clean, nil
	}
	return false, "", nil
}

func validateFiles(files []File, binary string) ([]File, error) {
	seen := make(map[string]bool, len(files))
	foundBinary := false
	for _, file := range files {
		if seen[file.Name] {
			return nil, fmt.Errorf("archive repeats %q", file.Name)
		}
		seen[file.Name] = true
		if file.Name == binary || file.Name == binary+".exe" {
			foundBinary = true
		}
	}
	if !foundBinary {
		return nil, fmt.Errorf("%s was not found in the release archive", binary)
	}
	return files, nil
}

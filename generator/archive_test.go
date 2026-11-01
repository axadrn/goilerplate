package generator

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

func TestWriteArchiveIsByteDeterministic(t *testing.T) {
	var first bytes.Buffer
	var second bytes.Buffer
	config := fullSQLiteHTMXConfig()
	if err := WriteArchive(&first, "3.0.0", config); err != nil {
		t.Fatalf("first WriteArchive() error = %v", err)
	}
	if err := WriteArchive(&second, "3.0.0", config); err != nil {
		t.Fatalf("second WriteArchive() error = %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("identical inputs produced different archives")
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(first.Bytes()))
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	tarReader := tar.NewReader(gzipReader)
	seenLock := false
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read archive: %v", err)
		}
		if header.Name == "goilerplate.lock" {
			seenLock = true
		}
	}
	if !seenLock {
		t.Fatal("archive does not contain goilerplate.lock")
	}
}

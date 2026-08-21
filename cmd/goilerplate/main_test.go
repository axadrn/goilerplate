package main

import (
	"os"
	"testing"
)

func TestIsTerminalRejectsNonTerminalFiles(t *testing.T) {
	regular, err := os.CreateTemp(t.TempDir(), "input")
	if err != nil {
		t.Fatal(err)
	}
	defer regular.Close()

	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()

	for name, file := range map[string]*os.File{
		"regular file": regular,
		"null device":  null,
		"pipe":         reader,
	} {
		t.Run(name, func(t *testing.T) {
			if isTerminal(file) {
				t.Fatalf("%s was classified as a terminal", name)
			}
		})
	}
}

package main

import (
	"os"
	"strings"
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

func TestBuildVersionUsesInjectedReleaseVersion(t *testing.T) {
	previous := version
	version = "v3.0.0-beta.1"
	t.Cleanup(func() { version = previous })
	if got := buildVersion(); got != "v3.0.0-beta.1" {
		t.Fatalf("buildVersion() = %q", got)
	}
}

func TestModuleUsesV3Path(t *testing.T) {
	content, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(content), "module github.com/axadrn/goilerplate/v3\n") {
		t.Fatalf("go.mod = %q", content)
	}
}

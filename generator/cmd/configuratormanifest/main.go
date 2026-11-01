package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/axadrn/goilerplate/v4/generator"
)

func main() {
	output := flag.String("out", "", "manifest output path")
	check := flag.Bool("check", false, "fail if the existing manifest differs")
	flag.Parse()
	if *output == "" {
		fmt.Fprintln(os.Stderr, "-out is required")
		os.Exit(2)
	}

	manifest, err := generator.BuildConfiguratorManifest("configurator-preview")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	content, err := json.Marshal(manifest)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	content = append(content, '\n')

	if *check {
		existing, err := os.ReadFile(*output)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if !bytes.Equal(existing, content) {
			fmt.Fprintln(os.Stderr, "configurator manifest is stale. Run go run ./generator/cmd/configuratormanifest -out <file>")
			os.Exit(1)
		}
		return
	}
	if err := os.WriteFile(*output, content, 0o644); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(os.Stderr, "manifest output directory does not exist")
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

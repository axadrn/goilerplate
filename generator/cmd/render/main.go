package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/axadrn/goilerplate/v4/generator"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	output := flag.String("out", "", "empty destination directory")
	configPath := flag.String("config", "", "path to generator config JSON")
	version := flag.String("version", "", "template version recorded in goilerplate.lock")
	flag.Parse()
	if *output == "" || *configPath == "" || *version == "" {
		return fmt.Errorf("out, config, and version are required")
	}
	content, err := os.ReadFile(*configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	var config generator.Config
	if err := json.Unmarshal(content, &config); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}
	return generator.Generate(*output, *version, config)
}

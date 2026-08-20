package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Config struct {
	APIURL       string `json:"api_url"`
	SessionToken string `json:"session_token,omitempty"`
}

type Store struct {
	path string
}

func DefaultStore() (*Store, error) {
	directory, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("find user configuration directory: %w", err)
	}
	return NewStore(filepath.Join(directory, "goilerplate", "config.json")), nil
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load() (Config, error) {
	content, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read CLI configuration: %w", err)
	}
	var configuration Config
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&configuration); err != nil {
		return Config{}, fmt.Errorf("decode CLI configuration: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Config{}, errors.New("CLI configuration must contain one JSON object")
	}
	return configuration, nil
}

func (s *Store) Save(configuration Config) error {
	directory := filepath.Dir(s.path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create CLI configuration directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return fmt.Errorf("secure CLI configuration directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create CLI configuration: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("secure CLI configuration: %w", err)
	}
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(configuration); err != nil {
		temporary.Close()
		return fmt.Errorf("write CLI configuration: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync CLI configuration: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close CLI configuration: %w", err)
	}
	if err := os.Rename(temporaryPath, s.path); err != nil {
		return fmt.Errorf("publish CLI configuration: %w", err)
	}
	return nil
}

package logger

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"testing"
)

func TestErrorsSplitsJoinedErrors(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	joined := errors.Join(errors.New("APP_ENV is invalid"), errors.New("APP_URL is invalid"))
	Errors("application stopped", fmt.Errorf("invalid configuration: %w", joined))

	var messages []string
	scanner := bufio.NewScanner(&output)
	for scanner.Scan() {
		var event struct {
			Message string `json:"msg"`
			Error   string `json:"error"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatal(err)
		}
		if event.Message != "application stopped" {
			t.Fatalf("message = %q", event.Message)
		}
		messages = append(messages, event.Error)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || messages[0] != "APP_ENV is invalid" || messages[1] != "APP_URL is invalid" {
		t.Fatalf("errors = %#v", messages)
	}
}

func TestLoadOwnsLoggerEnvironment(t *testing.T) {
	t.Setenv("LOG_LEVEL", " warn ")
	t.Setenv("LOG_FORMAT", " json ")
	t.Setenv("SENTRY_DSN", " https://examplePublicKey@example.com/1 ")

	cfg := Load(true)
	if cfg.Level != "warn" || cfg.Format != "json" {
		t.Fatalf("Load() logging values = %#v", cfg)
	}
	if cfg.SentryDSN != "https://examplePublicKey@example.com/1" {
		t.Fatalf("Load() SentryDSN = %q", cfg.SentryDSN)
	}
	if !cfg.IsDevelopment {
		t.Fatal("Load() lost the environment mode")
	}
}

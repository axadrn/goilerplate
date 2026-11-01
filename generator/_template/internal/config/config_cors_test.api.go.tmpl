package config

import (
	"slices"
	"strings"
	"testing"
)

func TestLoadCORSAllowedOrigins(t *testing.T) {
	setCORSTestEnvironment(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", " https://App.Example.com/ , http://localhost:5173,https://app.example.com")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://app.example.com", "http://localhost:5173"}
	if !slices.Equal(cfg.CORSAllowedOrigins, want) {
		t.Fatalf("CORSAllowedOrigins = %#v, want %#v", cfg.CORSAllowedOrigins, want)
	}
}

func TestLoadCORSAllowedOriginsDefaultsToNone(t *testing.T) {
	setCORSTestEnvironment(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.CORSAllowedOrigins) != 0 {
		t.Fatalf("CORSAllowedOrigins = %#v, want none", cfg.CORSAllowedOrigins)
	}
}

func TestLoadRejectsInvalidCORSAllowedOrigins(t *testing.T) {
	for _, value := range []string{
		"*",
		"app.example.com",
		"https://app.example.com/path",
		"https://app.example.com?x=1",
		"https://*.example.com",
		"ftp://app.example.com",
		"https://user@app.example.com",
		"https://app.example.com,",
	} {
		t.Run(value, func(t *testing.T) {
			setCORSTestEnvironment(t)
			t.Setenv("CORS_ALLOWED_ORIGINS", value)

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), "CORS_ALLOWED_ORIGINS") {
				t.Fatalf("Load() error = %v, want CORS_ALLOWED_ORIGINS error", err)
			}
		})
	}
}

func setCORSTestEnvironment(t *testing.T) {
	t.Helper()
	setCoreEnvironment(t)
}

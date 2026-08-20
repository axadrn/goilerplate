package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestStoreSavesAndLoadsSecureConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	store := NewStore(path)
	want := Config{APIURL: "https://goilerplate.com", SessionToken: "secret-token"}
	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("Load() = %#v, want %#v", got, want)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %o, want 600", fileInfo.Mode().Perm())
	}
	directoryInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && directoryInfo.Mode().Perm() != 0o700 {
		t.Fatalf("directory mode = %o, want 700", directoryInfo.Mode().Perm())
	}
}

func TestStoreReplacesExistingConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(path)
	if err := store.Save(Config{SessionToken: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(Config{SessionToken: "second"}); err != nil {
		t.Fatal(err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got.SessionToken != "second" {
		t.Fatalf("session token = %q, want second", got.SessionToken)
	}
}

func TestStoreRejectsUnknownConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"api_url":"https://goilerplate.com","mystery":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore(path).Load(); err == nil {
		t.Fatal("unknown configuration field was accepted")
	}
}

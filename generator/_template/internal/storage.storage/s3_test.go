package storage

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestS3StorageUsesSelectedBucket(t *testing.T) {
	var mu sync.Mutex
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	store, err := NewS3Storage(S3Config{
		Region: "us-east-1", Bucket: "test-bucket", AccessKey: "test-key", SecretKey: "test-secret",
		Endpoint: server.URL, PresignExpiryPublic: time.Hour, PresignExpiryPrivate: time.Minute,
	})
	if err != nil {
		t.Fatalf("NewS3Storage() error = %v", err)
	}
	if err := store.Save("users/avatar.png", strings.NewReader("image")); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := store.Delete("users/avatar.png"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if got := store.URL("users/avatar.png"); got != server.URL+"/test-bucket/users/avatar.png" {
		t.Fatalf("URL() = %q", got)
	}
	if got := store.GetPresignExpiryPrivate(); got != time.Minute {
		t.Fatalf("GetPresignExpiryPrivate() = %v", got)
	}

	mu.Lock()
	defer mu.Unlock()
	for _, want := range []string{"HEAD /test-bucket", "PUT /test-bucket/users/avatar.png", "DELETE /test-bucket/users/avatar.png"} {
		if !containsRequest(requests, want) {
			t.Fatalf("requests = %#v, missing %q", requests, want)
		}
	}
}

func TestS3StoragePresignsPrivateFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	store, err := NewS3Storage(S3Config{
		Region: "us-east-1", Bucket: "test-bucket", AccessKey: "test-key", SecretKey: "test-secret", Endpoint: server.URL,
	})
	if err != nil {
		t.Fatalf("NewS3Storage() error = %v", err)
	}
	url, err := store.PresignedURL("private/report.pdf", 5*time.Minute)
	if err != nil {
		t.Fatalf("PresignedURL() error = %v", err)
	}
	if !strings.HasPrefix(url, server.URL+"/test-bucket/private/report.pdf?") || !strings.Contains(url, "X-Amz-Signature=") {
		t.Fatalf("PresignedURL() = %q", url)
	}
}

func containsRequest(requests []string, want string) bool {
	for _, request := range requests {
		if request == want {
			return true
		}
	}
	return false
}

package utils

import (
	"sync"
	"testing"
)

// Run with -race: the first CN calls of a fresh process must not race
// while tailwind-merge-go builds its merger. Keep this test free of any
// warm-up call so it exercises that first build.
func TestCNIsSafeForConcurrentFirstUse(t *testing.T) {
	var start, done sync.WaitGroup
	start.Add(1)
	results := make([]string, 8)
	for index := range results {
		done.Add(1)
		go func() {
			defer done.Done()
			start.Wait()
			results[index] = CN("px-2 py-1", "px-4")
		}()
	}
	start.Done()
	done.Wait()
	for _, result := range results {
		if result != "py-1 px-4" {
			t.Fatalf("CN() = %q, want %q", result, "py-1 px-4")
		}
	}
}

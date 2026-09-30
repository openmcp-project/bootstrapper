package ocm_cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGetComponentVersionUsesCache proves a cached entry short-circuits before
// the `ocm` exec: the reference is bogus, so a cache miss would shell out and
// error. A hit must return the stored value with no error.
func TestGetComponentVersionUsesCache(t *testing.T) {
	ref := "example.invalid/repo//test-component:v1.2.3"
	key := NoOcmConfig + "\x00" + ref

	var want ComponentVersion
	want.Component.Name = "test-component"
	want.Repository = "example.invalid/repo"
	cvCache.Store(key, want)
	t.Cleanup(func() { cvCache.Delete(key) })

	got, err := GetComponentVersion(context.Background(), ref, NoOcmConfig)
	if err != nil {
		t.Fatalf("cache hit expected, got error (did it exec ocm?): %v", err)
	}
	if got.Component.Name != "test-component" {
		t.Fatalf("wrong cached value: got %q", got.Component.Name)
	}

	// Mutating the returned pointer must not corrupt the cache (values, not aliases).
	got.Component.Name = "mutated"
	again, err := GetComponentVersion(context.Background(), ref, NoOcmConfig)
	if err != nil {
		t.Fatalf("second cache hit errored: %v", err)
	}
	if again.Component.Name != "test-component" {
		t.Fatalf("cache aliasing: returned pointer shares cache state, got %q", again.Component.Name)
	}
}

// TestGetComponentVersionSkipsExecOnHit proves the cache eliminates the `ocm`
// exec on repeat resolution: a fake `ocm` on PATH records each invocation, and
// two GetComponentVersion calls for the same reference must exec it exactly once.
func TestGetComponentVersionSkipsExecOnHit(t *testing.T) {
	dir := t.TempDir()
	countFile := filepath.Join(dir, "count")

	// Fake `ocm`: append a line per call, print a minimal valid componentversion.
	script := "#!/bin/sh\necho x >> " + countFile + "\ncat <<'YAML'\ncomponent:\n  name: fake-component\n  version: v0.0.1\nYAML\n"
	if err := os.WriteFile(filepath.Join(dir, "ocm"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	ref := "example.invalid/repo//skip-exec-component:v0.0.1"
	cvCache.Delete(NoOcmConfig + "\x00" + ref)
	t.Cleanup(func() { cvCache.Delete(NoOcmConfig + "\x00" + ref) })

	for i := 0; i < 3; i++ {
		cv, err := GetComponentVersion(context.Background(), ref, NoOcmConfig)
		if err != nil {
			t.Fatalf("call %d errored: %v", i, err)
		}
		if cv.Component.Name != "fake-component" {
			t.Fatalf("call %d wrong value: %q", i, cv.Component.Name)
		}
	}

	data, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatalf("first call never exec'd ocm: %v", err)
	}
	if n := strings.Count(string(data), "\n"); n != 1 {
		t.Fatalf("expected 1 ocm exec across 3 calls, got %d: %q", n, data)
	}
}

package ocm_cli

import (
	"context"
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

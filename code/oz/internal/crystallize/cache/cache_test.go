package cache_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joaoajmatos/oz/internal/crystallize/cache"
)

func TestCache_MissOnEmpty(t *testing.T) {
	c := cache.New(filepath.Join(t.TempDir(), ".oz", "crystallize-cache.json"))
	_, ok := c.Get("notes/foo.md", []byte("content"), "model-a")
	if ok {
		t.Fatal("expected cache miss on empty cache")
	}
}

func TestCache_HitAfterSet(t *testing.T) {
	dir := t.TempDir()
	c := cache.New(filepath.Join(dir, ".oz", "crystallize-cache.json"))

	want := cache.Entry{Type: "adr", Confidence: "high", Title: "Auth Rewrite", Reason: "test", Source: "llm", Model: "model-a"}
	c.Set("notes/foo.md", []byte("content"), "model-a", want)

	got, ok := c.Get("notes/foo.md", []byte("content"), "model-a")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCache_MissOnContentChange(t *testing.T) {
	dir := t.TempDir()
	c := cache.New(filepath.Join(dir, ".oz", "crystallize-cache.json"))

	c.Set("notes/foo.md", []byte("original content"), "model-a", cache.Entry{Type: "adr"})

	_, ok := c.Get("notes/foo.md", []byte("changed content"), "model-a")
	if ok {
		t.Fatal("expected cache miss after content change")
	}
}

func TestCache_MissOnModelChange(t *testing.T) {
	dir := t.TempDir()
	c := cache.New(filepath.Join(dir, ".oz", "crystallize-cache.json"))

	c.Set("notes/foo.md", []byte("content"), "model-a", cache.Entry{Type: "adr"})

	_, ok := c.Get("notes/foo.md", []byte("content"), "model-b")
	if ok {
		t.Fatal("expected cache miss after model change")
	}
}

func TestCache_PersistAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".oz", "crystallize-cache.json")

	// Write and save.
	c1 := cache.New(path)
	want := cache.Entry{Type: "spec", Confidence: "high", Title: "My Spec", Reason: "test", Source: "llm", Model: "model-a"}
	c1.Set("notes/bar.md", []byte("spec content"), "model-a", want)
	if err := c1.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reload from disk.
	c2 := cache.New(path)
	got, ok := c2.Get("notes/bar.md", []byte("spec content"), "model-a")
	if !ok {
		t.Fatal("expected cache hit after reload")
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCache_SaveNoDirtyNoWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".oz", "crystallize-cache.json")

	c := cache.New(path)
	if err := c.Save(); err != nil {
		t.Fatalf("Save on clean cache: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected no file written when cache is not dirty")
	}
}

func TestCache_CorruptFileStartsFresh(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".oz", "crystallize-cache.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not valid json {{{"), 0o644); err != nil {
		t.Fatal(err)
	}

	c := cache.New(path)
	_, ok := c.Get("notes/foo.md", []byte("content"), "model-a")
	if ok {
		t.Fatal("expected miss after corrupt cache file")
	}
}

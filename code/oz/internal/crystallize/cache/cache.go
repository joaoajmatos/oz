// Package cache persists LLM classification results to disk, keyed by a hash
// of the note's path, content, and model. This makes repeated oz crystallize
// --dry-run calls deterministic: the same note with the same content produces
// the same classification without a new LLM call.
//
// The cache is stored at .oz/crystallize-cache.json in the workspace root.
// Entries are invalidated automatically when note content changes (the key
// includes a SHA-256 of the content).
package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Entry is a single cached classification result.
type Entry struct {
	Type       string `json:"type"`
	Confidence string `json:"confidence"`
	Title      string `json:"title"`
	Reason     string `json:"reason"`
	Source     string `json:"source"`
	Model      string `json:"model"`
}

// Cache is a thread-safe in-memory cache backed by a JSON file on disk.
type Cache struct {
	path    string
	mu      sync.Mutex
	entries map[string]Entry // key → entry
	dirty   bool
}

// New creates a Cache backed by the file at path. The file is read lazily on
// the first Get or Set call; a missing file is treated as an empty cache.
func New(path string) *Cache {
	return &Cache{
		path:    path,
		entries: nil, // loaded on demand
	}
}

// Get returns the cached Entry for the given note path, content, and model.
// ok is false if no valid cache entry exists.
func (c *Cache) Get(notePath string, content []byte, model string) (entry Entry, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.loadLocked(); err != nil {
		return Entry{}, false
	}
	key := cacheKey(notePath, content, model)
	e, ok := c.entries[key]
	return e, ok
}

// Set stores a classification result in the cache. Call Save to persist to disk.
func (c *Cache) Set(notePath string, content []byte, model string, e Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.loadLocked(); err != nil {
		c.entries = make(map[string]Entry)
	}
	key := cacheKey(notePath, content, model)
	c.entries[key] = e
	c.dirty = true
}

// Save writes the current cache to disk. It creates the parent directory if
// needed. Save is a no-op when the cache has not been modified.
func (c *Cache) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.dirty {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	data, err := json.MarshalIndent(c.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}
	if err := os.WriteFile(c.path, data, 0o644); err != nil {
		return fmt.Errorf("write cache: %w", err)
	}
	c.dirty = false
	return nil
}

// loadLocked reads the cache file from disk into c.entries. Must be called
// with c.mu held. A missing file initialises an empty cache without error.
func (c *Cache) loadLocked() error {
	if c.entries != nil {
		return nil
	}
	c.entries = make(map[string]Entry)
	data, err := os.ReadFile(c.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read cache: %w", err)
	}
	if err := json.Unmarshal(data, &c.entries); err != nil {
		// Corrupt cache: start fresh.
		c.entries = make(map[string]Entry)
	}
	return nil
}

// cacheKey returns a stable string key for a note path + content + model tuple.
func cacheKey(notePath string, content []byte, model string) string {
	h := sha256.New()
	h.Write([]byte(notePath))
	h.Write([]byte("\x00"))
	h.Write(content)
	h.Write([]byte("\x00"))
	h.Write([]byte(model))
	return fmt.Sprintf("%x", h.Sum(nil))
}

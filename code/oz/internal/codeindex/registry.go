package codeindex

import "sync"

var (
	mu       sync.RWMutex
	registry []LanguagePackage
)

// Register adds a LanguagePackage to the global registry. Duplicate Language()
// values are silently ignored. Intended to be called from package init() so that
// importing a language package is sufficient to activate it.
func Register(pkg LanguagePackage) {
	mu.Lock()
	defer mu.Unlock()
	for _, existing := range registry {
		if existing.Language() == pkg.Language() {
			return
		}
	}
	registry = append(registry, pkg)
}

// Detect calls Detect(root) on every registered LanguagePackage and returns
// those with Confidence > 0, in registration order.
func Detect(root string) []LanguagePackage {
	mu.RLock()
	defer mu.RUnlock()
	var active []LanguagePackage
	for _, pkg := range registry {
		if pkg.Detect(root).Confidence > 0 {
			active = append(active, pkg)
		}
	}
	return active
}

// All returns a copy of every registered LanguagePackage regardless of Detect
// result. Intended for tests.
func All() []LanguagePackage {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]LanguagePackage, len(registry))
	copy(out, registry)
	return out
}

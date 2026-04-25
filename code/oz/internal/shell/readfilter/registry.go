package readfilter

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/joaoajmatos/oz/internal/codeindex"
)

var (
	registryMu  sync.RWMutex
	byName      = map[string]LanguageReader{}
	byExtension = map[string]LanguageReader{}
)

func Register(reader LanguageReader) {
	if reader == nil {
		return
	}
	name := strings.TrimSpace(strings.ToLower(reader.Name()))
	if name == "" {
		return
	}

	registryMu.Lock()
	defer registryMu.Unlock()
	byName[name] = reader
	for _, ext := range reader.Extensions() {
		normalized := normalizeExtension(ext)
		if normalized == "" {
			continue
		}
		byExtension[normalized] = reader
	}
}

func Resolve(path string) LanguageReader {
	ext := normalizeExtension(filepath.Ext(path))
	registryMu.RLock()
	if direct := byExtension[ext]; direct != nil {
		registryMu.RUnlock()
		return direct
	}
	registryMu.RUnlock()

	if ext != "" {
		if language := codeindex.LanguageByExt(ext); language != "" {
			registryMu.RLock()
			fromLanguage := byName[strings.ToLower(language)]
			registryMu.RUnlock()
			if fromLanguage != nil {
				return fromLanguage
			}
		}
	}

	registryMu.RLock()
	fallback := byName["generic"]
	registryMu.RUnlock()
	if fallback != nil {
		return fallback
	}
	return passthroughReader{}
}

type passthroughReader struct{}

func (passthroughReader) Name() string { return "generic" }
func (passthroughReader) Extensions() []string {
	return nil
}
func (passthroughReader) Filter(content string, _ Options) (string, error) {
	return content, nil
}

func normalizeExtension(ext string) string {
	ext = strings.TrimSpace(strings.ToLower(ext))
	if ext == "" {
		return ""
	}
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return ext
}

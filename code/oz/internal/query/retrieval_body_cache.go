package query

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/joaoajmatos/oz/internal/graph"
)

type retrievalBodyCache struct {
	mu           sync.RWMutex
	graphHash    string
	tokenByBlock map[string][]string
}

var bodyTokenCache = &retrievalBodyCache{
	tokenByBlock: make(map[string][]string),
}

// ResetRetrievalBodyCacheForBenchmark clears the in-process retrieval body cache.
// Used by benchmarks to approximate cold-cache query runs.
func ResetRetrievalBodyCacheForBenchmark() {
	bodyTokenCache.mu.Lock()
	defer bodyTokenCache.mu.Unlock()
	bodyTokenCache.graphHash = ""
	bodyTokenCache.tokenByBlock = make(map[string][]string)
}

func loadRetrievalBodyTokens(workspacePath, graphHash string, n graph.Node, useBigrams bool) []string {
	if workspacePath == "" {
		return nil
	}
	// code_package: use aggregate package doc only (not an entire .go file body).
	if n.Type == graph.NodeTypeCodePackage {
		cacheKey := n.ID + "|pkgdoc|bigrams=" + boolString(useBigrams)
		if tokens, ok := bodyTokenCache.get(graphHash, cacheKey); ok {
			return tokens
		}
		tokens := TokenizeMulti(n.DocComment, useBigrams)
		bodyTokenCache.set(graphHash, cacheKey, tokens)
		return tokens
	}
	if n.File == "" {
		return nil
	}
	cacheKey := n.ID + "|bigrams=" + boolString(useBigrams)
	if tokens, ok := bodyTokenCache.get(graphHash, cacheKey); ok {
		return tokens
	}

	raw, err := os.ReadFile(filepath.Join(workspacePath, n.File))
	if err != nil {
		return nil
	}
	body := extractRetrievalBody(string(raw), n)
	tokens := TokenizeMulti(body, useBigrams)
	bodyTokenCache.set(graphHash, cacheKey, tokens)
	return tokens
}

func (c *retrievalBodyCache) get(graphHash, key string) ([]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.graphHash != graphHash {
		return nil, false
	}
	tokens, ok := c.tokenByBlock[key]
	if !ok {
		return nil, false
	}
	return append([]string(nil), tokens...), true
}

func (c *retrievalBodyCache) set(graphHash, key string, tokens []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.graphHash != graphHash {
		c.graphHash = graphHash
		c.tokenByBlock = make(map[string][]string)
	}
	c.tokenByBlock[key] = append([]string(nil), tokens...)
}

func extractRetrievalBody(content string, n graph.Node) string {
	// Decision, context snapshot, and notes are whole-file bodies.
	if n.Type == graph.NodeTypeDecision || n.Type == graph.NodeTypeContextSnapshot || n.Type == graph.NodeTypeNote {
		return strings.TrimSpace(content)
	}
	// spec_section / doc nodes should use their section body only.
	if n.Section == "" {
		return strings.TrimSpace(content)
	}
	if section, ok := markdownSectionBody(content, n.Section); ok {
		return section
	}
	return strings.TrimSpace(content)
}

func markdownSectionBody(content, heading string) (string, bool) {
	var currentHeading string
	var body strings.Builder
	target := strings.TrimSpace(heading)
	scanner := bufio.NewScanner(strings.NewReader(content))
	found := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "## ") {
			if found {
				break
			}
			currentHeading = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			if currentHeading == target {
				found = true
			}
			continue
		}
		if !found {
			continue
		}
		if strings.TrimSpace(line) == "---" {
			continue
		}
		body.WriteString(line)
		body.WriteByte('\n')
	}
	if !found {
		return "", false
	}
	return strings.TrimSpace(body.String()), true
}

func boolString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}


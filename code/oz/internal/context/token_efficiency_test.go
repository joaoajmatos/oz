package context_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/query"
	"github.com/joaoajmatos/oz/internal/testws"
)

// TestTokenEfficiency measures the token efficiency of oz context query output
// relative to a full workspace read. S7-02 target: ≤ 10% of full workspace size.
//
// Token count is approximated as word count (sufficient for ratio comparison).
func TestTokenEfficiency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping token efficiency test in short mode")
	}

	ws := testws.FromFixture(t, "../query/testdata/golden/02_medium/workspace.yaml").Build()

	// Representative queries covering a range of agents.
	queries := []struct {
		text          string
		expectedAgent string
	}{
		{"implement REST API endpoint for user authentication", "backend"},
		{"add OAuth login button to the React login page", "frontend"},
		{"set up Kubernetes deployment for the API service", "infra"},
		{"design a new colour token for error states", "design"},
		{"write an end-to-end Cypress test for the checkout flow", "qa"},
		{"triage a CVE in the JWT library", "security"},
		{"add a TOTP MFA flow to the login screen", "auth"},
		{"build an ETL pipeline to sync events to the data warehouse", "data"},
		{"update the API reference with the new pagination field", "docs-agent"},
		{"release the iOS app to the App Store", "mobile"},
	}

	// Measure full workspace read size.
	fullSize := workspaceTotalSize(t, ws.Path())
	if fullSize == 0 {
		t.Fatal("full workspace size is 0 — workspace not built correctly")
	}

	var totalQuerySize int
	for _, q := range queries {
		result := query.Run(ws.Path(), q.text)
		encoded, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("marshal result for %q: %v", q.text, err)
		}
		totalQuerySize += len(encoded)
	}

	avgQuerySize := totalQuerySize / len(queries)
	ratio := float64(avgQuerySize) / float64(fullSize) * 100

	t.Logf("S7-02 Token Efficiency")
	t.Logf("  Full workspace size:   %d bytes (~%d tokens)", fullSize, estimateTokens(fullSize))
	t.Logf("  Avg query output size: %d bytes (~%d tokens)", avgQuerySize, estimateTokens(avgQuerySize))
	t.Logf("  Ratio:                 %.1f%% of full workspace (target: ≤10%%)", ratio)

	const targetPct = 10.0
	if ratio > targetPct {
		t.Errorf("query output is %.1f%% of full workspace read; target ≤%.0f%%", ratio, targetPct)
	}
}

// workspaceTotalSize returns the total byte count of all markdown files in the workspace.
func workspaceTotalSize(t *testing.T, root string) int {
	t.Helper()
	var total int
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		total += int(info.Size())
		return nil
	})
	if err != nil {
		t.Fatalf("walk workspace: %v", err)
	}
	return total
}

// estimateTokens approximates token count from byte count (4 bytes ≈ 1 token).
func estimateTokens(bytes int) int {
	return bytes / 4
}

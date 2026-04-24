package query_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/joaoajmatos/oz/internal/query"
	"github.com/joaoajmatos/oz/internal/testws"
)

func TestBuildRawQueryDebug_includesRetrievalMath(t *testing.T) {
	t.Parallel()
	corpusDir := filepath.Join("testdata", "golden", "04_retrieval")
	ws := testws.FromFixture(t, filepath.Join(corpusDir, "workspace.yaml")).Build()

	raw := query.BuildRawQueryDebug(ws.Path(), "how is drift detection implemented in the audit package", query.Options{})

	if len(raw.Retrieval) == 0 {
		t.Fatal("expected non-empty raw.retrieval for --raw")
	}
	first := raw.Retrieval[0]
	if first.File == "" {
		t.Fatal("expected file on first retrieval row")
	}
	b, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	var round query.RawQueryDebug
	if err := json.Unmarshal(b, &round); err != nil {
		t.Fatalf("round-trip JSON: %v", err)
	}
	if len(round.Retrieval) != len(raw.Retrieval) {
		t.Fatalf("retrieval len after json round-trip: %d vs %d", len(round.Retrieval), len(raw.Retrieval))
	}
}

package track

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestStoreInsertAndQuery(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer closeStore(t, store)

	now := time.Now().Unix()
	mustInsert(t, store, Run{Command: "first", RecordedAt: now - 20, DurationMs: 1, TokenBefore: 10, TokenAfter: 8, TokenSaved: 2, ReductionPct: 20, MatchedFilter: "generic", ExitCode: 0})
	mustInsert(t, store, Run{Command: "second", RecordedAt: now - 10, DurationMs: 1, TokenBefore: 10, TokenAfter: 7, TokenSaved: 3, ReductionPct: 30, MatchedFilter: "generic", ExitCode: 0})
	mustInsert(t, store, Run{Command: "third", RecordedAt: now - 5, DurationMs: 1, TokenBefore: 10, TokenAfter: 6, TokenSaved: 4, ReductionPct: 40, MatchedFilter: "generic", ExitCode: 1})

	runs, err := store.Query(QueryOpts{})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(runs))
	}
	if runs[0].Command != "third" || runs[1].Command != "second" || runs[2].Command != "first" {
		t.Fatalf("unexpected order: %#v", []string{runs[0].Command, runs[1].Command, runs[2].Command})
	}
}

func TestStoreQueryLimit(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer closeStore(t, store)

	now := time.Now().Unix()
	for i := 0; i < 5; i++ {
		mustInsert(t, store, Run{
			Command:       "cmd",
			RecordedAt:    now + int64(i),
			DurationMs:    1,
			TokenBefore:   10,
			TokenAfter:    9,
			TokenSaved:    1,
			ReductionPct:  10,
			MatchedFilter: "generic",
			ExitCode:      0,
		})
	}

	runs, err := store.Query(QueryOpts{Limit: 3})
	if err != nil {
		t.Fatalf("query with limit: %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("expected 3 runs, got %d", len(runs))
	}
}

func TestStorePrune(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer closeStore(t, store)

	now := time.Now()
	oldRecordedAt := now.AddDate(0, 0, -90).Unix()
	recentRecordedAt := now.AddDate(0, 0, -1).Unix()

	mustInsert(t, store, Run{Command: "old", RecordedAt: oldRecordedAt, DurationMs: 1, TokenBefore: 10, TokenAfter: 9, TokenSaved: 1, ReductionPct: 10, MatchedFilter: "generic", ExitCode: 0})
	mustInsert(t, store, Run{Command: "recent", RecordedAt: recentRecordedAt, DurationMs: 1, TokenBefore: 10, TokenAfter: 9, TokenSaved: 1, ReductionPct: 10, MatchedFilter: "generic", ExitCode: 0})

	if err := store.Prune(30); err != nil {
		t.Fatalf("prune: %v", err)
	}

	runs, err := store.Query(QueryOpts{})
	if err != nil {
		t.Fatalf("query after prune: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run after prune, got %d", len(runs))
	}
	if runs[0].Command != "recent" {
		t.Fatalf("expected recent run to remain, got %q", runs[0].Command)
	}
}

func TestStoreDefaultPath(t *testing.T) {
	t.Parallel()

	originalXDG := os.Getenv("XDG_DATA_HOME")
	t.Cleanup(func() {
		if originalXDG == "" {
			_ = os.Unsetenv("XDG_DATA_HOME")
		} else {
			_ = os.Setenv("XDG_DATA_HOME", originalXDG)
		}
	})

	custom := filepath.Join(t.TempDir(), "xdg")
	if err := os.Setenv("XDG_DATA_HOME", custom); err != nil {
		t.Fatalf("set XDG_DATA_HOME: %v", err)
	}
	if got, want := DefaultPath(), filepath.Join(custom, "oz", "shell-track.db"); got != want {
		t.Fatalf("DefaultPath with XDG_DATA_HOME=%q, want %q", got, want)
	}

	if err := os.Unsetenv("XDG_DATA_HOME"); err != nil {
		t.Fatalf("unset XDG_DATA_HOME: %v", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("home dir: %v", err)
	}
	if got, want := DefaultPath(), filepath.Join(home, ".local", "share", "oz", "shell-track.db"); got != want {
		t.Fatalf("DefaultPath fallback=%q, want %q", got, want)
	}
}

func TestStoreQuerySinceDaysBoundary(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer closeStore(t, store)

	now := time.Unix(1700000000, 0)
	cutoff := now.AddDate(0, 0, -30).Unix()
	mustInsert(t, store, Run{Command: "before", RecordedAt: cutoff - 1, DurationMs: 1, TokenBefore: 10, TokenAfter: 9, TokenSaved: 1, ReductionPct: 10, MatchedFilter: "generic", ExitCode: 0})
	mustInsert(t, store, Run{Command: "at-cutoff", RecordedAt: cutoff, DurationMs: 1, TokenBefore: 10, TokenAfter: 9, TokenSaved: 1, ReductionPct: 10, MatchedFilter: "generic", ExitCode: 0})
	mustInsert(t, store, Run{Command: "after", RecordedAt: cutoff + 1, DurationMs: 1, TokenBefore: 10, TokenAfter: 9, TokenSaved: 1, ReductionPct: 10, MatchedFilter: "generic", ExitCode: 0})

	runs, err := store.QuerySinceDays(30, now)
	if err != nil {
		t.Fatalf("QuerySinceDays: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}
	if runs[0].Command != "after" || runs[1].Command != "at-cutoff" {
		t.Fatalf("unexpected retained runs: %#v", []string{runs[0].Command, runs[1].Command})
	}
}

func TestStoreConcurrentInsertAndQuery(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	defer closeStore(t, store)

	now := time.Now().Unix()
	var wg sync.WaitGroup
	errCh := make(chan error, 40)
	for i := 0; i < 20; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			errCh <- store.Insert(Run{
				Command:       "c",
				RecordedAt:    now + int64(i),
				DurationMs:    1,
				TokenBefore:   4,
				TokenAfter:    2,
				TokenSaved:    2,
				ReductionPct:  50,
				MatchedFilter: "generic",
				ExitCode:      0,
			})
			_, queryErr := store.Query(QueryOpts{Limit: 5})
			errCh <- queryErr
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent store operation failed: %v", err)
		}
	}

	runs, err := store.Query(QueryOpts{})
	if err != nil {
		t.Fatalf("final query: %v", err)
	}
	if len(runs) != 20 {
		t.Fatalf("expected 20 runs after concurrent inserts, got %d", len(runs))
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "track.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}

func closeStore(t *testing.T, store *Store) {
	t.Helper()
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}
}

func mustInsert(t *testing.T, store *Store, run Run) {
	t.Helper()
	if err := store.Insert(run); err != nil {
		t.Fatalf("insert run: %v", err)
	}
}

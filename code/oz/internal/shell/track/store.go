package track

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func DefaultPath() string {
	xdg := os.Getenv("XDG_DATA_HOME")
	if strings.TrimSpace(xdg) != "" {
		return filepath.Join(xdg, "oz", "shell-track.db")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".local", "share", "oz", "shell-track.db")
	}
	return filepath.Join(home, ".local", "share", "oz", "shell-track.db")
}

func Open(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configure sqlite busy timeout: %w", err)
	}

	if _, err := db.Exec(createRunsTable); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create runs table: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Insert(r Run) error {
	if s == nil || s.db == nil {
		err := fmt.Errorf("store is not open")
		log.Printf("shell track insert skipped: %v", err)
		return err
	}

	_, err := s.db.Exec(
		`INSERT INTO runs (
			command, recorded_at, duration_ms, token_before, token_after,
			token_saved, reduction_pct, matched_filter, exit_code
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Command, r.RecordedAt, r.DurationMs, r.TokenBefore, r.TokenAfter,
		r.TokenSaved, r.ReductionPct, r.MatchedFilter, r.ExitCode,
	)
	if err != nil {
		log.Printf("shell track insert failed: %v", err)
	}
	return err
}

func (s *Store) Query(opts QueryOpts) ([]Run, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("store is not open")
	}

	args := make([]any, 0, 3)
	clauses := []string{"1=1"}
	if opts.Since > 0 {
		clauses = append(clauses, "recorded_at >= ?")
		args = append(args, opts.Since)
	}
	if opts.Command != "" {
		clauses = append(clauses, "command LIKE ?")
		args = append(args, "%"+opts.Command+"%")
	}

	query := `SELECT id, command, recorded_at, duration_ms, token_before, token_after, token_saved, reduction_pct, matched_filter, exit_code FROM runs WHERE ` +
		strings.Join(clauses, " AND ") +
		` ORDER BY recorded_at DESC, id DESC`
	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query runs: %w", err)
	}
	defer rows.Close()

	var runs []Run
	for rows.Next() {
		var r Run
		if err := rows.Scan(
			&r.ID,
			&r.Command,
			&r.RecordedAt,
			&r.DurationMs,
			&r.TokenBefore,
			&r.TokenAfter,
			&r.TokenSaved,
			&r.ReductionPct,
			&r.MatchedFilter,
			&r.ExitCode,
		); err != nil {
			return nil, fmt.Errorf("scan run: %w", err)
		}
		runs = append(runs, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate runs: %w", err)
	}
	return runs, nil
}

func (s *Store) QuerySinceDays(retentionDays int, now time.Time) ([]Run, error) {
	opts := QueryOpts{}
	if retentionDays > 0 {
		opts.Since = now.AddDate(0, 0, -retentionDays).Unix()
	}
	return s.Query(opts)
}

func (s *Store) Prune(retentionDays int) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("store is not open")
	}
	if retentionDays <= 0 {
		return nil
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays).Unix()
	_, err := s.db.Exec(`DELETE FROM runs WHERE recorded_at < ?`, cutoff)
	if err != nil {
		return fmt.Errorf("prune runs: %w", err)
	}
	return nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

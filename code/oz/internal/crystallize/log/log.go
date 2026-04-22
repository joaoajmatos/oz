// Package log provides an append-only crystallize log that records each note
// promotion with a timestamp, source path, target path, and artifact type.
//
// Log format (tab-separated):
//
//	<RFC3339>	<source>	→	<target>	[<type>]
//
// The log is stored at context/crystallize.log in the workspace root and
// created (along with any missing parent directories) on the first append.
package log

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Logger appends entries to a crystallize log file.
type Logger struct {
	path string
}

// New returns a Logger that writes to path.
func New(path string) *Logger {
	return &Logger{path: path}
}

// Append appends a single promotion entry to the log file. The file and its
// parent directory are created if they do not exist.
func (l *Logger) Append(source, target, artifactType string) error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	defer f.Close()
	entry := fmt.Sprintf("%s\t%s\t→\t%s\t[%s]\n",
		time.Now().UTC().Format(time.RFC3339), source, target, artifactType)
	if _, err := fmt.Fprint(f, entry); err != nil {
		return fmt.Errorf("write log entry: %w", err)
	}
	return nil
}

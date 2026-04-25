package tee

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func Write(command, stdout, stderr string) (*string, error) {
	baseDir := filepath.Join(os.TempDir(), "oz", "shell-tee")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create tee dir: %w", err)
	}

	name := fmt.Sprintf("run-%d.log", time.Now().UnixNano())
	path := filepath.Join(baseDir, name)
	content := fmt.Sprintf(
		"command: %s\n\n--- stdout ---\n%s\n\n--- stderr ---\n%s\n",
		command, stdout, stderr,
	)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("write tee artifact: %w", err)
	}
	return &path, nil
}

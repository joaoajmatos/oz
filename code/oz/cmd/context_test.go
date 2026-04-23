package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/semantic"
)

func TestFindWorkspaceRoot_FromNestedDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# AGENTS\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "OZ.md"), []byte("oz standard: v0.1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "code", "oz")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if chdirErr := os.Chdir(origWD); chdirErr != nil {
			t.Fatalf("restore working directory: %v", chdirErr)
		}
	}()

	if err := os.Chdir(nested); err != nil {
		t.Fatal(err)
	}

	got, err := findWorkspaceRoot()
	if err != nil {
		t.Fatalf("findWorkspaceRoot returned error: %v", err)
	}
	wantResolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	gotResolved, err := filepath.EvalSymlinks(got)
	if err != nil {
		t.Fatal(err)
	}
	if gotResolved != wantResolved {
		t.Fatalf("expected workspace root %q, got %q", wantResolved, gotResolved)
	}
}

func TestShouldSkipEnrich(t *testing.T) {
	cases := []struct {
		name           string
		existing       *semantic.Overlay
		graphHash      string
		requestedModel string
		force          bool
		want           bool
	}{
		{
			name:      "skip when fresh and no model override",
			existing:  &semantic.Overlay{GraphHash: "abc", Model: "m1"},
			graphHash: "abc",
			want:      true,
		},
		{
			name:           "do not skip when model override differs",
			existing:       &semantic.Overlay{GraphHash: "abc", Model: "m1"},
			graphHash:      "abc",
			requestedModel: "m2",
			want:           false,
		},
		{
			name:           "skip when requested model matches",
			existing:       &semantic.Overlay{GraphHash: "abc", Model: "m1"},
			graphHash:      "abc",
			requestedModel: "m1",
			want:           true,
		},
		{
			name:      "do not skip when stale",
			existing:  &semantic.Overlay{GraphHash: "old", Model: "m1"},
			graphHash: "new",
			want:      false,
		},
		{
			name:      "do not skip when no overlay",
			existing:  nil,
			graphHash: "abc",
			want:      false,
		},
		{
			name:      "do not skip when forced",
			existing:  &semantic.Overlay{GraphHash: "abc", Model: "m1"},
			graphHash: "abc",
			force:     true,
			want:      false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shouldSkipEnrich(tc.existing, tc.graphHash, tc.requestedModel, tc.force)
			if got != tc.want {
				t.Fatalf("shouldSkipEnrich() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestPrintContextServeBanner_PlainWhenWriterNotTTY(t *testing.T) {
	var buf bytes.Buffer
	printContextServeBanner(&buf, "/tmp/ws")
	out := buf.String()
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("expected plain text when writer is not an os.File TTY, got ANSI: %q", out)
	}
	for _, sub := range []string{
		"context serve — MCP stdio server",
		"workspace: /tmp/ws",
		"JSON-RPC on stdin/stdout",
		"Ctrl+C, or close stdin (EOF)",
		"stderr",
	} {
		if !strings.Contains(out, sub) {
			t.Fatalf("banner should contain %q, got:\n%s", sub, out)
		}
	}
}

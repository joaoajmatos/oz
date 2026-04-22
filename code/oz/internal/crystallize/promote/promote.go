// Package promote implements atomic promotion of classified notes to canonical
// workspace artifacts. All writes use a temp-file + rename pattern so any
// interrupted promotion leaves either the unchanged target or the complete new
// file — never a partial write.
//
// Artifact types and their target paths:
//
//	adr       → specs/decisions/NNNN-<slug>.md  (auto-numbered, template injected)
//	spec      → specs/<slug>.md
//	guide     → docs/guides/<slug>.md
//	arch      → docs/<slug>.md
//	open-item → docs/open-items.md  (append mode)
package promote

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Artifact type constants mirror classifier.ArtifactType without creating an
// import dependency on that package.
const (
	TypeADR      = "adr"
	TypeSpec     = "spec"
	TypeGuide    = "guide"
	TypeArch     = "arch"
	TypeOpenItem = "open-item"
)

// CollisionError is returned when the target path already exists.
// Existing holds the current file content so the caller can present a diff.
type CollisionError struct {
	Target   string
	Existing []byte
}

func (e *CollisionError) Error() string {
	return fmt.Sprintf("collision: %s already exists", e.Target)
}

// Result records the outcome of a successful promotion.
type Result struct {
	TargetPath string
	Appended   bool // true for open-item append mode
}

// Options configures a promotion.
type Options struct {
	// Today is used for ADR frontmatter date injection.
	// Defaults to time.Now() when zero.
	Today time.Time
}

// Promote writes content to its canonical location for the given artifact type.
// Returns *CollisionError if the target already exists (open-item always appends
// and never returns CollisionError).
func Promote(root, sourcePath string, content []byte, artifactType, title string, opts Options) (Result, error) {
	if opts.Today.IsZero() {
		opts.Today = time.Now()
	}
	switch artifactType {
	case TypeADR:
		return promoteADR(root, content, title, opts)
	case TypeSpec:
		return promoteToFile(filepath.Join(root, "specs"), content, title)
	case TypeGuide:
		return promoteToFile(filepath.Join(root, "docs", "guides"), content, title)
	case TypeArch:
		return promoteToFile(filepath.Join(root, "docs"), content, title)
	case TypeOpenItem:
		return appendOpenItem(root, content, title)
	default:
		return Result{}, fmt.Errorf("unsupported artifact type: %q", artifactType)
	}
}

// TargetPath returns the canonical target path for the given type and title
// without performing any writes. For ADR types it scans the decisions directory
// to determine the next number.
func TargetPath(root, artifactType, title string) (string, error) {
	switch artifactType {
	case TypeADR:
		n, err := ADRNumber(filepath.Join(root, "specs", "decisions"))
		if err != nil {
			return "", err
		}
		return adrPath(root, n, title), nil
	case TypeSpec:
		return filepath.Join(root, "specs", slugify(title)+".md"), nil
	case TypeGuide:
		return filepath.Join(root, "docs", "guides", slugify(title)+".md"), nil
	case TypeArch:
		return filepath.Join(root, "docs", slugify(title)+".md"), nil
	case TypeOpenItem:
		return filepath.Join(root, "docs", "open-items.md"), nil
	default:
		return "", fmt.Errorf("unsupported artifact type: %q", artifactType)
	}
}

// ADRNumber scans decisionsDir for existing ADR files matching the pattern
// NNNN-*.md and returns the next sequential number. Returns 1 if the directory
// does not exist or contains no numbered ADR files.
func ADRNumber(decisionsDir string) (int, error) {
	entries, err := os.ReadDir(decisionsDir)
	if errors.Is(err, os.ErrNotExist) {
		return 1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read decisions dir: %w", err)
	}
	re := regexp.MustCompile(`^(\d{4})-`)
	max := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := re.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[1])
		if n > max {
			max = n
		}
	}
	return max + 1, nil
}

// promoteADR promotes content as an ADR to specs/decisions/NNNN-<slug>.md.
func promoteADR(root string, content []byte, title string, opts Options) (Result, error) {
	decisionsDir := filepath.Join(root, "specs", "decisions")
	n, err := ADRNumber(decisionsDir)
	if err != nil {
		return Result{}, err
	}
	target := adrPath(root, n, title)
	if err := checkCollision(target); err != nil {
		return Result{}, err
	}
	injected := injectADRTemplate(content, n, title, opts.Today)
	if err := atomicWrite(target, injected); err != nil {
		return Result{}, err
	}
	return Result{TargetPath: target}, nil
}

// promoteToFile promotes content to dir/<slug>.md.
func promoteToFile(dir string, content []byte, title string) (Result, error) {
	target := filepath.Join(dir, slugify(title)+".md")
	if err := checkCollision(target); err != nil {
		return Result{}, err
	}
	if err := atomicWrite(target, content); err != nil {
		return Result{}, err
	}
	return Result{TargetPath: target}, nil
}

// appendOpenItem appends content as a new ## <title> section to
// docs/open-items.md, creating the file with a standard header if absent.
func appendOpenItem(root string, content []byte, title string) (Result, error) {
	target := filepath.Join(root, "docs", "open-items.md")

	existing, err := os.ReadFile(target)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Result{}, fmt.Errorf("read open-items: %w", err)
	}

	var buf strings.Builder
	if len(existing) == 0 {
		buf.WriteString("# Open Items\n\n")
	} else {
		buf.Write(existing)
		if !strings.HasSuffix(string(existing), "\n") {
			buf.WriteByte('\n')
		}
		buf.WriteByte('\n')
	}
	fmt.Fprintf(&buf, "## %s\n\n", title)
	buf.Write(content)
	if !strings.HasSuffix(string(content), "\n") {
		buf.WriteByte('\n')
	}

	if err := atomicWrite(target, []byte(buf.String())); err != nil {
		return Result{}, err
	}
	return Result{TargetPath: target, Appended: true}, nil
}

// checkCollision returns *CollisionError if the target path already exists.
func checkCollision(target string) error {
	existing, err := os.ReadFile(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check collision for %s: %w", target, err)
	}
	return &CollisionError{Target: target, Existing: existing}
}

// atomicWrite writes content to path using a temp-file + rename pattern.
// Any stale .tmp file from a previous interrupted run is cleaned up first.
// afterWriteHook, if non-nil, is called between the write and the rename —
// this is used only in tests to simulate an interrupted write.
func atomicWrite(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	tmp := path + ".tmp"
	// Clean up any stale .tmp from a previous interrupted run.
	_ = os.Remove(tmp)
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if afterWriteHook != nil {
		afterWriteHook()
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename to target: %w", err)
	}
	return nil
}

// afterWriteHook is called between tmp write and rename, used only in tests.
var afterWriteHook func()

// injectADRTemplate prepends ADR frontmatter and an H1 title to content.
func injectADRTemplate(content []byte, n int, title string, today time.Time) []byte {
	header := fmt.Sprintf("---\nstatus: accepted\ndate: %s\n---\n\n# ADR-%04d: %s\n\n",
		today.Format("2006-01-02"), n, title)
	return append([]byte(header), content...)
}

// adrPath returns the canonical path for ADR number n with the given title.
func adrPath(root string, n int, title string) string {
	return filepath.Join(root, "specs", "decisions",
		fmt.Sprintf("%04d-%s.md", n, slugify(title)))
}

// slugify converts a title to a lowercase hyphen-separated filename slug.
func slugify(title string) string {
	title = strings.ToLower(title)
	var buf strings.Builder
	prev := rune('-')
	for _, r := range title {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			buf.WriteRune(r)
			prev = r
		} else if prev != '-' {
			buf.WriteByte('-')
			prev = '-'
		}
	}
	return strings.Trim(buf.String(), "-")
}

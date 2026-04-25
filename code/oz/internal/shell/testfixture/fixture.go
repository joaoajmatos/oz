package testfixture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type Fixture struct{ dir string }

func New(dir string) *Fixture { return &Fixture{dir: dir} }

// Load reads <dir>/<name>/input.txt and <dir>/<name>/expected.txt.
// Calls t.Fatal on missing files.
func (f *Fixture) Load(t *testing.T, name string) (input, expected string) {
	t.Helper()

	base := filepath.Join(f.dir, FixtureName(name))
	inputPath := filepath.Join(base, "input.txt")
	expectedPath := filepath.Join(base, "expected.txt")

	inputBytes, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read fixture input %q: %v", inputPath, err)
	}
	expectedBytes, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read fixture expected %q: %v", expectedPath, err)
	}

	return string(inputBytes), string(expectedBytes)
}

// Assert compares actual vs <dir>/<name>/expected.txt.
// If UPDATE_GOLDEN=1, writes actual to the file instead.
func (f *Fixture) Assert(t *testing.T, name, actual string) {
	t.Helper()

	base := filepath.Join(f.dir, FixtureName(name))
	expectedPath := filepath.Join(base, "expected.txt")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(base, 0o755); err != nil {
			t.Fatalf("create fixture dir %q: %v", base, err)
		}
		if err := os.WriteFile(expectedPath, []byte(actual), 0o644); err != nil {
			t.Fatalf("write fixture expected %q: %v", expectedPath, err)
		}
		return
	}

	_, expected := f.Load(t, name)
	if actual != expected {
		t.Fatalf("fixture %q mismatch\nexpected:\n%s\nactual:\n%s", name, expected, actual)
	}
}

// LoadFixture loads from the package "testdata" directory.
func LoadFixture(t *testing.T, name string) (input, expected string) {
	t.Helper()
	return New("testdata").Load(t, name)
}

// AssertGolden asserts against "testdata/<name>/expected.txt".
func AssertGolden(t *testing.T, name, actual string) {
	t.Helper()
	New("testdata").Assert(t, name, actual)
}

// FixtureName sanitizes spaces and slashes to underscores.
func FixtureName(name string) string {
	sanitized := strings.ReplaceAll(name, " ", "_")
	sanitized = strings.ReplaceAll(sanitized, "/", "_")
	return sanitized
}

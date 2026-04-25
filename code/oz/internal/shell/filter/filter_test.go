package filter_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/shell/filter"
	"github.com/joaoajmatos/oz/internal/shell/testfixture"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args []string
		want filter.ID
	}{
		{args: []string{"git", "status"}, want: filter.FilterGitStatus},
		{args: []string{"git", "diff"}, want: filter.FilterGitDiff},
		{args: []string{"git", "log"}, want: filter.FilterGitLog},
		{args: []string{"git", "blame", "x.go"}, want: filter.FilterGitBlame},
		{args: []string{"git", "show", "HEAD"}, want: filter.FilterGitShow},
		{args: []string{"rg", "TODO"}, want: filter.FilterRG},
		{args: []string{"go", "test", "./..."}, want: filter.FilterGoTest},
		{args: []string{"go", "build", "./..."}, want: filter.FilterGoBuild},
		{args: []string{"go", "vet", "./..."}, want: filter.FilterGoVet},
		{args: []string{"staticcheck", "./..."}, want: filter.FilterGoVet},
		{args: []string{"ls", "-la"}, want: filter.FilterLs},
		{args: []string{"find", "."}, want: filter.FilterFind},
		{args: []string{"tree"}, want: filter.FilterTree},
		{args: []string{"jq", "."}, want: filter.FilterJSON},
		{args: []string{"make"}, want: filter.FilterMake},
		{args: []string{"npm", "install"}, want: filter.FilterNpm},
		{args: []string{"yarn", "install"}, want: filter.FilterNpm},
		{args: []string{"docker", "build", "."}, want: filter.FilterDocker},
		{args: []string{"curl", "-I", "https://example.com"}, want: filter.FilterHTTP},
		{args: []string{"env"}, want: filter.FilterEnv},
		{args: []string{"wc", "-l", "x"}, want: filter.FilterWc},
		{args: []string{"df", "-h"}, want: filter.FilterDf},
		{args: []string{"ps", "aux"}, want: filter.FilterPs},
		{args: []string{"top", "-b", "-n", "1"}, want: filter.FilterPs},
		{args: []string{"diff", "-u", "a", "b"}, want: filter.FilterDiff},
		{args: []string{"cargo", "build"}, want: filter.FilterCargo},
		{args: []string{"pytest", "-q"}, want: filter.FilterPytest},
		{args: []string{"python", "-m", "pytest"}, want: filter.FilterPytest},
		{args: []string{"echo", "hello"}, want: filter.FilterGeneric},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			t.Parallel()
			if got := filter.Classify(tc.args); got != tc.want {
				t.Fatalf("Classify(%v)=%q, want %q", tc.args, got, tc.want)
			}
		})
	}
}

func TestGoldenFixtures(t *testing.T) {
	t.Parallel()

	fixture := testfixture.New("testdata")
	cases := []struct {
		name string
		args []string
	}{
		{name: "git_status", args: []string{"git", "status"}},
		{name: "git_diff", args: []string{"git", "diff"}},
		{name: "rg", args: []string{"rg", "TODO"}},
		{name: "go_test", args: []string{"go", "test", "./..."}},
		{name: "ls", args: []string{"ls", "-la"}},
		{name: "find", args: []string{"find", "."}},
		{name: "tree", args: []string{"tree"}},
		{name: "json_jq", args: []string{"jq", "."}},
		{name: "json_sniff", args: []string{"true"}},
		{name: "git_log", args: []string{"git", "log"}},
		{name: "git_blame", args: []string{"git", "blame", "x.go"}},
		{name: "git_show", args: []string{"git", "show", "HEAD"}},
		{name: "go_build", args: []string{"go", "build", "./..."}},
		{name: "go_vet", args: []string{"go", "vet", "./..."}},
		{name: "staticcheck", args: []string{"staticcheck", "./..."}},
		{name: "make", args: []string{"make"}},
		{name: "cargo", args: []string{"cargo", "build"}},
		{name: "pytest", args: []string{"pytest", "-q"}},
		{name: "npm", args: []string{"npm", "install"}},
		{name: "docker", args: []string{"docker", "build", "."}},
		{name: "http", args: []string{"curl", "-i", "http://example.com"}},
		{name: "env", args: []string{"env"}},
		{name: "wc", args: []string{"wc", "README.md"}},
		{name: "df", args: []string{"df", "-h"}},
		{name: "ps", args: []string{"ps", "aux"}},
		{name: "top_batch", args: []string{"top", "-b", "-n", "1"}},
		{name: "diff", args: []string{"diff", "-u", "a", "b"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input, _ := fixture.Load(t, tc.name)
			out, errOut, _, err := filter.Apply(tc.args, input, "", 0, false)
			if err != nil {
				t.Fatalf("Apply failed: %v", err)
			}
			actual := strings.TrimSpace(out)
			if errOut != "" {
				actual += "\n\nstderr:\n" + strings.TrimSpace(errOut)
			}
			fixture.Assert(t, tc.name, actual+"\n")
		})
	}
}

func TestGoldenFailureFixtures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		args []string
	}{
		{name: "go_build_failure", args: []string{"go", "build", "./..."}},
		{name: "make_failure", args: []string{"make"}},
		{name: "cargo_failure", args: []string{"cargo", "build"}},
		{name: "pytest_failure", args: []string{"pytest", "-q"}},
		{name: "npm_failure", args: []string{"npm", "run", "build"}},
		{name: "docker_failure", args: []string{"docker", "build", "."}},
		{name: "http_failure", args: []string{"curl", "-i", "http://example.com"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			base := filepath.Join("testdata", testfixture.FixtureName(tc.name))
			stdoutBytes, err := os.ReadFile(filepath.Join(base, "stdout.txt"))
			if err != nil {
				t.Fatalf("read stdout: %v", err)
			}
			stderrBytes, err := os.ReadFile(filepath.Join(base, "stderr.txt"))
			if err != nil {
				t.Fatalf("read stderr: %v", err)
			}
			out, errOut, _, err := filter.Apply(tc.args, string(stdoutBytes), string(stderrBytes), 1, false)
			if err != nil {
				t.Fatalf("Apply failed: %v", err)
			}
			actual := strings.TrimSpace(out)
			if errOut != "" {
				actual += "\n\nstderr:\n" + strings.TrimSpace(errOut)
			}
			actual += "\n"
			expectedPath := filepath.Join(base, "expected.txt")
			expectedBytes, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("read expected %q: %v", expectedPath, err)
			}
			expected := string(expectedBytes)
			if os.Getenv("UPDATE_GOLDEN") == "1" {
				if err := os.WriteFile(expectedPath, []byte(actual), 0o644); err != nil {
					t.Fatalf("write golden %q: %v", expectedPath, err)
				}
				return
			}
			if actual != expected {
				t.Fatalf("fixture %q mismatch\nexpected:\n%s\nactual:\n%s", tc.name, expected, actual)
			}
		})
	}
}

func TestApplyFamilies(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		args       []string
		stdout     string
		stderr     string
		exitCode   int
		wantFilter filter.ID
		wantSubstr string
	}{
		{
			name:       "git_status",
			args:       []string{"git", "status"},
			stdout:     " M cmd/shell.go\n?? internal/shell/filter/filter.go\n",
			wantFilter: filter.FilterGitStatus,
			wantSubstr: "git status summary",
		},
		{
			name:       "git_diff",
			args:       []string{"git", "diff"},
			stdout:     "diff --git a/a.go b/a.go\n+++ b/a.go\n--- a/a.go\n 1 file changed, 2 insertions(+), 1 deletions(-)\n",
			wantFilter: filter.FilterGitDiff,
			wantSubstr: "git diff summary",
		},
		{
			name:       "rg",
			args:       []string{"rg", "TODO"},
			stdout:     "cmd/shell.go:10:TODO: add docs\ncmd/shell.go:22:TODO: tests\n",
			wantFilter: filter.FilterRG,
			wantSubstr: "cmd/shell.go",
		},
		{
			name:       "go_test",
			args:       []string{"go", "test", "./..."},
			stdout:     "--- FAIL: TestX (0.00s)\nFAIL\tgithub.com/example/pkg\t0.1s\n",
			exitCode:   1,
			wantFilter: filter.FilterGoTest,
			wantSubstr: "--- FAIL: TestX",
		},
		{
			name:       "json_sniff_apply",
			args:       []string{"true"},
			stdout:     `{"a":1}`,
			wantFilter: filter.FilterJSON,
			wantSubstr: "json object",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotStdout, _, gotFilter, err := filter.Apply(tc.args, tc.stdout, tc.stderr, tc.exitCode, false)
			if err != nil {
				t.Fatalf("Apply returned error: %v", err)
			}
			if gotFilter != tc.wantFilter {
				t.Fatalf("matched filter=%q, want %q", gotFilter, tc.wantFilter)
			}
			if !strings.Contains(gotStdout, tc.wantSubstr) {
				t.Fatalf("expected compact stdout to contain %q, got %q", tc.wantSubstr, gotStdout)
			}
		})
	}
}

package filter_test

import (
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/shell/filter"
	"github.com/joaoajmatos/oz/internal/shell/testfixture"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"git", "status"}, want: filter.FilterGitStatus},
		{args: []string{"git", "diff"}, want: filter.FilterGitDiff},
		{args: []string{"rg", "TODO"}, want: filter.FilterRG},
		{args: []string{"go", "test", "./..."}, want: filter.FilterGoTest},
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

func TestApplyFamilies(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		args       []string
		stdout     string
		stderr     string
		exitCode   int
		wantFilter string
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

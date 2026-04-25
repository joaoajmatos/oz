package filter_test

import (
	"testing"

	"github.com/joaoajmatos/oz/internal/shell/filter"
	"github.com/joaoajmatos/oz/internal/shell/testfixture"
)

func TestDeterminismFromFixtures(t *testing.T) {
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
			out1, err1, _, err := filter.Apply(tc.args, input, "", 0, false)
			if err != nil {
				t.Fatalf("first run failed: %v", err)
			}
			out2, err2, _, err := filter.Apply(tc.args, input, "", 0, false)
			if err != nil {
				t.Fatalf("second run failed: %v", err)
			}
			if out1 != out2 || err1 != err2 {
				t.Fatalf("non-deterministic output for %s", tc.name)
			}
		})
	}
}

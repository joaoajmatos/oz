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

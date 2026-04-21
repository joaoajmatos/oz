package query_test

import (
    "fmt"
    "testing"
    "github.com/joaoajmatos/oz/internal/query"
    "github.com/joaoajmatos/oz/internal/testws"
)

func TestDiagnose(t *testing.T) {
	suites, _ := testws.LoadGoldenSuites(t, "testdata/golden")
	for _, suite := range suites {
		ws := suite.Build(t)
		for _, q := range suite.Queries {
			result := query.Run(ws.Path(), q.Query)
			ok := q.Matches(result)
			if !ok {
				fmt.Printf("[%s] FAIL: %q\n  → got=%q(%.2f) want=%q\n",
					suite.Name, q.Query, result.Agent, result.Confidence, q.ExpectedAgent)
			}
		}
	}
}

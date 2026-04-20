package cmd

import (
	"github.com/oz-tools/oz/internal/audit"
	"github.com/oz-tools/oz/internal/audit/drift"
	"github.com/spf13/cobra"
)

var (
	driftIncludeTests bool
	driftIncludeDocs  bool
)

var auditDriftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Check for spec-code drift: missing paths, renamed symbols, undocumented exports",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runSingleCheck(cmd, &drift.Check{}, audit.Options{
			IncludeTests: driftIncludeTests,
			IncludeDocs:  driftIncludeDocs,
		})
	},
}

func init() {
	auditDriftCmd.Flags().BoolVar(&driftIncludeTests, "include-tests", false, "include _test.go symbols in drift extraction")
	auditDriftCmd.Flags().BoolVar(&driftIncludeDocs, "include-docs", false, "also scan docs/ for code references")
	registerCheck(&drift.Check{})
}

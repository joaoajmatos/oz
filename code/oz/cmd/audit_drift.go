package cmd

import (
	"github.com/joaoajmatos/oz/internal/audit"
	"github.com/joaoajmatos/oz/internal/audit/drift"
	"github.com/spf13/cobra"
)

var auditDriftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Check for spec-code drift: missing paths, renamed symbols, undocumented exports",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runSingleCheck(cmd, &drift.Check{}, audit.Options{
			IncludeTests: auditIncludeDriftTests,
			IncludeDocs:  auditIncludeDriftDocs,
		})
	},
}

func init() {
	registerCheck(&drift.Check{})
}

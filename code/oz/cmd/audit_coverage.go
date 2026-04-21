package cmd

import (
	"github.com/joaoajmatos/oz/internal/audit"
	"github.com/joaoajmatos/oz/internal/audit/coverage"
	"github.com/spf13/cobra"
)

var auditCoverageCmd = &cobra.Command{
	Use:   "coverage",
	Short: "Check for unowned code directories and dangling scope paths",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runSingleCheck(cmd, &coverage.Check{}, audit.Options{})
	},
}

func init() {
	registerCheck(&coverage.Check{})
}

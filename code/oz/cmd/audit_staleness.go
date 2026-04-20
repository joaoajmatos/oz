package cmd

import (
	"github.com/oz-tools/oz/internal/audit/staleness"
	"github.com/spf13/cobra"
)

var auditStalenessCmd = &cobra.Command{
	Use:   "staleness",
	Short: "Check for stale graph.json or semantic.json",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runSingleCheck(cmd, &staleness.Check{})
	},
}

func init() {
	registerCheck(&staleness.Check{})
}

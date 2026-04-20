package cmd

import (
	"github.com/oz-tools/oz/internal/audit/orphans"
	"github.com/spf13/cobra"
)

var auditOrphansCmd = &cobra.Command{
	Use:   "orphans",
	Short: "Check for orphaned specs, decisions, docs, and context snapshots",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runSingleCheck(cmd, &orphans.Check{})
	},
}

func init() {
	registerCheck(&orphans.Check{})
}

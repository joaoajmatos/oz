package cmd

import (
	"github.com/joaoajmatos/oz/internal/audit"
	"github.com/joaoajmatos/oz/internal/audit/orphans"
	"github.com/spf13/cobra"
)

var auditOrphansCmd = &cobra.Command{
	Use:   "orphans",
	Short: "Check for orphaned specs, decisions, docs, and context snapshots",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runSingleCheck(cmd, &orphans.Check{}, audit.Options{})
	},
}

func init() {
	registerCheck(&orphans.Check{})
}

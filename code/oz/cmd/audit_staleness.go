package cmd

import (
	"github.com/oz-tools/oz/internal/audit"
	"github.com/spf13/cobra"
)

type stalenessCheck struct{}

func (c *stalenessCheck) Name() string { return "staleness" }
func (c *stalenessCheck) Codes() []string {
	return []string{"STALE001", "STALE002", "STALE003"}
}
func (c *stalenessCheck) Run(_ string, _ audit.Options) ([]audit.Finding, error) { return nil, nil }

var auditStalenessCmd = &cobra.Command{
	Use:   "staleness",
	Short: "Check for stale context snapshots and notes (stub)",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runSingleCheck(cmd, &stalenessCheck{})
	},
}

func init() {
	registerCheck(&stalenessCheck{})
}

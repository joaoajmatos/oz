package cmd

import (
	"github.com/oz-tools/oz/internal/audit"
	"github.com/spf13/cobra"
)

type orphansCheck struct{}

func (c *orphansCheck) Name() string { return "orphans" }
func (c *orphansCheck) Codes() []string {
	return []string{"ORPH001", "ORPH002", "ORPH003"}
}
func (c *orphansCheck) Run(_ string, _ audit.Options) ([]audit.Finding, error) { return nil, nil }

var auditOrphansCmd = &cobra.Command{
	Use:   "orphans",
	Short: "Check for orphaned agents, specs, and decisions (stub)",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runSingleCheck(cmd, &orphansCheck{})
	},
}

func init() {
	registerCheck(&orphansCheck{})
}

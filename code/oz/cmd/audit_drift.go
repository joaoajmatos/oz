package cmd

import (
	"github.com/oz-tools/oz/internal/audit"
	"github.com/spf13/cobra"
)

type driftCheck struct{}

func (c *driftCheck) Name() string { return "drift" }
func (c *driftCheck) Codes() []string {
	return []string{"DRIFT001", "DRIFT002", "DRIFT003"}
}
func (c *driftCheck) Run(_ string, _ audit.Options) ([]audit.Finding, error) { return nil, nil }

var auditDriftCmd = &cobra.Command{
	Use:   "drift",
	Short: "Check for code/spec drift in agent scope declarations (stub)",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runSingleCheck(cmd, &driftCheck{})
	},
}

func init() {
	registerCheck(&driftCheck{})
}

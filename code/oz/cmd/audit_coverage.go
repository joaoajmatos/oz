package cmd

import (
	"github.com/oz-tools/oz/internal/audit"
	"github.com/spf13/cobra"
)

type coverageCheck struct{}

func (c *coverageCheck) Name() string { return "coverage" }
func (c *coverageCheck) Codes() []string {
	return []string{"COV001", "COV002", "COV003"}
}
func (c *coverageCheck) Run(_ string, _ audit.Options) ([]audit.Finding, error) { return nil, nil }

var auditCoverageCmd = &cobra.Command{
	Use:   "coverage",
	Short: "Check for spec sections and decisions without agent coverage (stub)",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runSingleCheck(cmd, &coverageCheck{})
	},
}

func init() {
	registerCheck(&coverageCheck{})
}

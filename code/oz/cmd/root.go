package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "oz",
	Short: "oz — workspace convention and toolset for LLM-first development",
	Long: `oz gives any LLM a structured, predictable workspace it can immediately
understand — without custom integrations or provider-specific configuration.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(auditCmd)
}

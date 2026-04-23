package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "oz",
	Short: "oz — workspace convention and toolset for LLM-first development",
	Long: `oz gives any LLM a structured, predictable workspace it can immediately
understand — with clean integrations for Claude Code, Cursor, and any other editor or model.`,
	// Show the banner when oz is run with no subcommand.
	RunE: func(cmd *cobra.Command, args []string) error {
		PrintBanner()
		fmt.Println("  " + styleBrand.Render("oz") + "  " + styleSubtle.Render("workspace convention and toolset for LLM-first development"))
		fmt.Println()
		fmt.Println("  " + styleCmd.Render("oz tipz") + "  " + styleSubtle.Render(randomOzTip()))
		fmt.Println()
		fmt.Println(styleSubtle.Render("  Run") + " " + styleCmd.Render("oz --help") + " " + styleSubtle.Render("for available commands."))
		fmt.Println()
		return nil
	},
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
	rootCmd.AddCommand(crystallizeCmd)
	rootCmd.AddCommand(repairCmd)
	installBrandedHelp(rootCmd)
}

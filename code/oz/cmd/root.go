package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/joaoajmatos/oz/internal/termstyle"
)

var rootCmd = &cobra.Command{
	Use:   "oz",
	Short: "oz — workspace convention and toolset for LLM-first development",
	Long: `oz gives any LLM a structured, predictable workspace it can immediately
understand — with clean integrations for Claude Code, Cursor, and any other editor or model.`,
	// Show the banner when oz is run with no subcommand.
	RunE: func(cmd *cobra.Command, args []string) error {
		PrintBanner()
		fmt.Println("  " + termstyle.Brand.Render("oz") + "  " + termstyle.Subtle.Render("workspace convention and toolset for LLM-first development"))
		fmt.Println()
		fmt.Println("  " + termstyle.Command.Render("oz tipz") + "  " + termstyle.Subtle.Render(randomOzTip()))
		fmt.Println()
		fmt.Println(termstyle.Subtle.Render("  Run") + " " + termstyle.Command.Render("oz --help") + " " + termstyle.Subtle.Render("for available commands."))
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
	rootCmd.AddCommand(shellCmd)
	installBrandedHelp(rootCmd)
}

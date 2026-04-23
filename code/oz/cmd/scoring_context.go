package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/joaoajmatos/oz/internal/query"
	"github.com/joaoajmatos/oz/internal/termstyle"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	scoringShowJSON     bool
	scoringShowDefaults bool
	scoringDescribeJSON bool
	scoringNoColor      bool
)

var contextScoringCmd = &cobra.Command{
	Use:   "scoring",
	Short: "Inspect and edit context/scoring.toml (BM25F and routing)",
	Long: `Commands to list keys, show effective values, get or set one key, describe tuning,
and validate the TOML file. See "oz context scoring list" for all keys.`,
}

var contextScoringShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print effective scoring configuration",
	RunE:  runScoringShow,
}

var contextScoringGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Print the current value for one key (section.name)",
	Args:  cobra.ExactArgs(1),
	RunE:  runScoringGet,
}

var contextScoringSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set one key and rewrite context/scoring.toml",
	Args:  cobra.ExactArgs(2),
	RunE:  runScoringSet,
}

var contextScoringListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tunable keys with one-line descriptions",
	RunE:  runScoringList,
}

var contextScoringDescribeCmd = &cobra.Command{
	Use:   "describe <key>",
	Short: "Long-form help, default, and current value for one key",
	Args:  cobra.ExactArgs(1),
	RunE:  runScoringDescribe,
}

var contextScoringValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Check for unknown keys and invalid values in context/scoring.toml",
	RunE:  runScoringValidate,
}

// contextDescribeAliasCmd is a shortcut: `oz context describe` === `oz context scoring describe`
// (users often type the former and expect it to work).
var contextDescribeAliasCmd = &cobra.Command{
	Use:   "describe <key>",
	Short: "Describe one scoring.toml key (alias for `oz context scoring describe`)",
	Long: `Long-form help, default, and current value for one key in context/scoring.toml.
Same as: oz context scoring describe <key>`,
	Args: cobra.ExactArgs(1),
	RunE: runScoringDescribe,
}

func init() {
	contextCmd.AddCommand(contextDescribeAliasCmd)
	contextCmd.AddCommand(contextScoringCmd)
	contextScoringCmd.AddCommand(contextScoringShowCmd)
	contextScoringCmd.AddCommand(contextScoringGetCmd)
	contextScoringCmd.AddCommand(contextScoringSetCmd)
	contextScoringCmd.AddCommand(contextScoringListCmd)
	contextScoringCmd.AddCommand(contextScoringDescribeCmd)
	contextScoringCmd.AddCommand(contextScoringValidateCmd)

	contextScoringShowCmd.Flags().BoolVar(&scoringShowJSON, "json", false, "print as JSON")
	contextScoringShowCmd.Flags().BoolVar(&scoringShowDefaults, "defaults", false, "print built-in defaults only (ignore file)")
	contextScoringDescribeCmd.Flags().BoolVar(&scoringDescribeJSON, "json", false, "print as JSON")
	contextDescribeAliasCmd.Flags().BoolVar(&scoringDescribeJSON, "json", false, "print as JSON")
	contextDescribeAliasCmd.Flags().BoolVar(&scoringNoColor, "no-color", false, "disable ANSI colors")
	contextScoringCmd.PersistentFlags().BoolVar(&scoringNoColor, "no-color", false, "disable ANSI colors")
}

func runScoringShow(_ *cobra.Command, _ []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	cfg := query.LoadConfig(root)
	if scoringShowDefaults {
		cfg = query.DefaultScoringConfig()
	}
	if scoringShowJSON {
		return printScoringConfigJSON(cfg)
	}
	keys := allScoringKeysSorted()
	if useScoringTTY() {
		fmt.Println(termstyle.Brand.Render("context/scoring — effective config"))
		for _, k := range keys {
			s, err := query.GetScoringValueString(cfg, k)
			if err != nil {
				return err
			}
			fmt.Printf("  %s  %s\n", termstyle.Subtle.Render(k+":"), termstyle.AccentBold.Render(s))
		}
		return nil
	}
	for _, k := range keys {
		s, err := query.GetScoringValueString(cfg, k)
		if err != nil {
			return err
		}
		fmt.Printf("%s\t%s\n", k, s)
	}
	return nil
}

func runScoringGet(_ *cobra.Command, args []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	key := args[0]
	if _, ok := query.ScoringKeyMetaByName(key); !ok {
		return fmt.Errorf("unknown key %q — run: oz context scoring list", key)
	}
	s, err := query.GetScoringValueString(query.LoadConfig(root), key)
	if err != nil {
		return err
	}
	fmt.Println(s)
	return nil
}

func runScoringSet(_ *cobra.Command, args []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	if err := query.SetScoringKey(root, args[0], args[1]); err != nil {
		return err
	}
	if useScoringTTY() {
		fmt.Fprintf(os.Stderr, "%s %s\n", termstyle.OK.Render("✓"), termstyle.Muted.Render("wrote "+query.ScoringTOMLPath(root)))
		return nil
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", query.ScoringTOMLPath(root))
	return nil
}

func runScoringList(_ *cobra.Command, _ []string) error {
	lines := make([]query.ScoringKeyMeta, len(query.AllScoringKeyMeta))
	copy(lines, query.AllScoringKeyMeta)
	sort.Slice(lines, func(i, j int) bool { return lines[i].Key < lines[j].Key })
	if useScoringTTY() {
		fmt.Println(termstyle.Brand.Render("Tunable keys (section.name)"))
		for _, meta := range lines {
			fmt.Printf("  %s\n    %s\n", termstyle.AccentBold.Render(meta.Key), termstyle.Muted.Render(meta.Title))
		}
		return nil
	}
	for _, meta := range lines {
		fmt.Printf("%s\n  %s\n", meta.Key, meta.Title)
	}
	return nil
}

func runScoringDescribe(_ *cobra.Command, args []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	key := args[0]
	if _, ok := query.ScoringKeyMetaByName(key); !ok {
		return fmt.Errorf("unknown key %q — run: oz context scoring list", key)
	}
	d, err := query.BuildScoringDescribe(root, key)
	if err != nil {
		return err
	}
	if scoringDescribeJSON {
		b, err := json.MarshalIndent(d, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	}
	if useScoringTTY() {
		fmt.Println(termstyle.Brand.Render(d.Title))
		fmt.Println(termstyle.Section.Render(d.Key))
		fmt.Printf("  %s %s\n", termstyle.Subtle.Render("type:"), d.Type)
		fmt.Printf("  %s %s\n", termstyle.Subtle.Render("default:"), termstyle.AccentBold.Render(d.Default))
		fmt.Printf("  %s %s\n", termstyle.Subtle.Render("current:"), termstyle.AccentBold.Render(d.Current))
		fmt.Println()
		for _, line := range strings.Split(strings.TrimSpace(d.Description), "\n") {
			fmt.Println(termstyle.Muted.Render(line))
		}
		return nil
	}
	fmt.Printf("key:     %s\n", d.Key)
	fmt.Printf("type:    %s\ndefault: %s\ncurrent: %s\n", d.Type, d.Default, d.Current)
	fmt.Printf("\n%s\n", d.Description)
	return nil
}

func runScoringValidate(_ *cobra.Command, _ []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}
	if err := query.ValidateScoringFile(root); err != nil {
		return err
	}
	if useScoringTTY() {
		p := query.ScoringTOMLPath(root)
		if _, statErr := os.Stat(p); os.IsNotExist(statErr) {
			fmt.Println(termstyle.Muted.Render("no context/scoring.toml — defaults apply; nothing to validate"))
			return nil
		}
		fmt.Fprintf(os.Stdout, "%s %s\n", termstyle.OK.Render("✓"), termstyle.Muted.Render(p+" is valid"))
		return nil
	}
	if _, statErr := os.Stat(query.ScoringTOMLPath(root)); os.IsNotExist(statErr) {
		fmt.Println("ok (no file; defaults apply)")
		return nil
	}
	fmt.Println("ok")
	return nil
}

func allScoringKeysSorted() []string {
	var keys []string
	for k := range scoringKeySet() {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func scoringKeySet() map[string]struct{} {
	m := make(map[string]struct{})
	for _, meta := range query.AllScoringKeyMeta {
		m[meta.Key] = struct{}{}
	}
	return m
}

func printScoringConfigJSON(cfg query.ScoringConfig) error {
	out := make(map[string]string, len(query.AllScoringKeyMeta))
	for _, meta := range query.AllScoringKeyMeta {
		s, err := query.GetScoringValueString(cfg, meta.Key)
		if err != nil {
			return err
		}
		out[meta.Key] = s
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func useScoringTTY() bool {
	if scoringNoColor || os.Getenv("NO_COLOR") != "" {
		return false
	}
	return isatty.IsTerminal(os.Stdout.Fd())
}

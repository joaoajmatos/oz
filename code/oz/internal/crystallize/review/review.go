// Package review implements the interactive per-file diff review for oz crystallize.
//
// This package uses charmbracelet/huh (an existing project dependency) for
// prompts, while diffs are rendered as plain unified diff text to Options.Out.
//
// Flow:
//   - For ≤15 files: ReviewItem is called for each candidate.
//   - For >15 files: BatchSummary is called first to choose per-type bulk actions
//     (review individually / bulk accept / bulk skip).
//
// All output goes to Options.Out (defaults to os.Stdout); all input comes from
// Options.In (defaults to os.Stdin). This makes the package fully testable.
package review

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/joaoajmatos/oz/internal/termstyle"
	"github.com/mattn/go-isatty"
)

// Action is the user's decision for a single note.
type Action string

const (
	// ActionAccept means the user confirmed the promotion.
	ActionAccept Action = "accept"
	// ActionEdit means the user wants to edit the note content before re-reviewing.
	ActionEdit Action = "edit"
	// ActionSkip means the user wants to skip this note for now.
	ActionSkip Action = "skip"
	// ActionQuit means the user wants to stop the entire review session.
	// No further notes will be presented; already-promoted files are kept.
	ActionQuit Action = "quit"
)

// Decision is the result of reviewing a single item.
type Decision struct {
	Action Action
	// Source is the (possibly edited) note content to be promoted.
	Source []byte
}

// BatchChoice is the bulk action chosen for an artifact type in batch mode.
type BatchChoice string

const (
	BatchReview BatchChoice = "review"
	BatchAccept BatchChoice = "accept"
	BatchSkip   BatchChoice = "skip"
)

// Item is a classified note ready for review.
type Item struct {
	SourcePath   string // relative path from workspace root
	TargetPath   string // relative canonical target path
	ArtifactType string
	Title        string
	Confidence   string
	Reason       string
	Source       []byte // note content (possibly edited)
	Proposed     []byte // content that would be written (after template injection)
}

// Options configures a review session.
type Options struct {
	// AcceptAll marks every item as accepted without prompting.
	AcceptAll bool
	// DryRun shows diffs but returns ActionSkip for every item.
	DryRun bool
	// Out is where review output is written. Defaults to os.Stdout.
	Out io.Writer
	// In is the source of user input. Defaults to os.Stdin.
	In io.Reader
	// Theme is the huh theme used for prompts. Defaults to huh.ThemeCharm().
	Theme *huh.Theme
	// Color enables ANSI color output for diffs. Defaults to true when Out is a TTY.
	Color bool
}

func (o *Options) defaults() {
	if o.Out == nil {
		o.Out = os.Stdout
	}
	if o.In == nil {
		o.In = os.Stdin
	}
	if o.Theme == nil {
		o.Theme = huh.ThemeCharm()
	}
	// Default to color when writing to an interactive terminal.
	if f, ok := o.Out.(*os.File); ok && isatty.IsTerminal(f.Fd()) {
		o.Color = true
	}
}

// ReviewItem presents a diff for a single classified note and prompts for an
// action. idx and total are used for the "[n/N]" progress indicator.
func ReviewItem(item Item, idx, total int, opts Options) (Decision, error) {
	opts.defaults()
	w := opts.Out

	fmt.Fprintf(w, "\n[%d/%d] %s\n", idx, total, item.SourcePath)
	fmt.Fprintf(w, "      → %s\n", item.TargetPath)
	fmt.Fprintf(w, "      type: %-12s  confidence: %s\n", item.ArtifactType, item.Confidence)
	if item.Reason != "" {
		fmt.Fprintf(w, "      reason: %s\n", item.Reason)
	}
	fmt.Fprintln(w)
	printDiff(w, item.SourcePath, item.TargetPath, item.Source, item.Proposed, opts.Color)

	if opts.DryRun {
		return Decision{Action: ActionSkip, Source: item.Source}, nil
	}
	if opts.AcceptAll {
		return Decision{Action: ActionAccept, Source: item.Source}, nil
	}

	action, err := promptAction(opts)
	if err != nil {
		return Decision{}, err
	}
	if action != ActionEdit {
		return Decision{Action: action, Source: item.Source}, nil
	}

	edited := string(item.Source)
	editForm := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Edit note content (will be re-classified into the proposed artifact)").
				Value(&edited),
		),
	).WithInput(opts.In).WithOutput(opts.Out).WithTheme(opts.Theme)

	if err := editForm.Run(); err != nil {
		return Decision{}, err
	}
	return Decision{Action: ActionEdit, Source: []byte(edited)}, nil
}

// BatchPlan is the result of BatchSummary.
type BatchPlan struct {
	ByType map[string]BatchChoice
}

// BatchSummary prints the full classification table and asks the user to choose
// per-type bulk actions. It is called when there are more than 15 classifiable notes.
func BatchSummary(items []Item, opts Options) (BatchPlan, error) {
	opts.defaults()
	w := opts.Out

	fmt.Fprintf(w, "\n%d files classified:\n\n", len(items))
	printTable(w, items)
	fmt.Fprintln(w)

	present := typesPresent(items)
	plan := BatchPlan{ByType: make(map[string]BatchChoice)}

	var choices []struct {
		t      string
		choice *BatchChoice
	}
	var fields []huh.Field
	for _, t := range orderedTypes(present) {
		c := BatchReview
		tt := t
		choices = append(choices, struct {
			t      string
			choice *BatchChoice
		}{t: tt, choice: &c})
		fields = append(fields, huh.NewSelect[BatchChoice]().
			Title(fmt.Sprintf("%s (%d)", tt, present[tt])).
			Options(
				huh.NewOption("Review individually", BatchReview),
				huh.NewOption("Bulk accept", BatchAccept),
				huh.NewOption("Bulk skip", BatchSkip),
			).
			Value(&c).
			Description("Choose what to do with this artifact type."),
		)
	}
	form := huh.NewForm(huh.NewGroup(fields...)).
		WithInput(opts.In).
		WithOutput(opts.Out).
		WithTheme(opts.Theme)
	if err := form.Run(); err != nil {
		return BatchPlan{}, err
	}
	for _, c := range choices {
		plan.ByType[c.t] = *c.choice
	}
	return plan, nil
}

// PrintTable prints a classification summary table to w.
func PrintTable(w io.Writer, items []Item) {
	printTable(w, items)
}

// printDiff renders a unified diff, optionally colorized.
func printDiff(w io.Writer, source, target string, old, new []byte, color bool) {
	diff := unifiedDiff(source, target, old, new)
	if color {
		fmt.Fprint(w, colorizeUnifiedDiff(diff))
	} else {
		fmt.Fprint(w, diff)
	}
	fmt.Fprintln(w)
}

// printTable renders a fixed-width classification table.
func printTable(w io.Writer, items []Item) {
	const fileW = 44
	fmt.Fprintf(w, "  %-*s  %-12s  %s\n", fileW, "FILE", "TYPE", "CONFIDENCE")
	fmt.Fprintln(w, "  "+strings.Repeat("─", fileW+28))
	for _, item := range items {
		name := item.SourcePath
		if len(name) > fileW {
			name = "..." + name[len(name)-(fileW-3):]
		}
		fmt.Fprintf(w, "  %-*s  %-12s  %s\n", fileW, name, item.ArtifactType, item.Confidence)
	}
}

func promptAction(opts Options) (Action, error) {
	var action Action
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[Action]().
				Title("Action").
				Options(
					huh.NewOption("Accept", ActionAccept),
					huh.NewOption("Edit", ActionEdit),
					huh.NewOption("Skip", ActionSkip),
					huh.NewOption("Quit", ActionQuit),
				).
				Value(&action),
		),
	).WithInput(opts.In).WithOutput(opts.Out).WithTheme(opts.Theme)
	if err := form.Run(); err != nil {
		return ActionQuit, err
	}
	return action, nil
}

func unifiedDiff(source, target string, old, new []byte) string {
	oldLines := splitLines(old)
	newLines := splitLines(new)

	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n", source)
	fmt.Fprintf(&b, "+++ %s\n", target)
	fmt.Fprintf(&b, "@@ -1,%d +1,%d @@\n", len(oldLines), len(newLines))
	for _, l := range oldLines {
		fmt.Fprintf(&b, "-%s\n", l)
	}
	for _, l := range newLines {
		fmt.Fprintf(&b, "+%s\n", l)
	}
	return b.String()
}

func splitLines(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	s := strings.TrimRight(string(b), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func colorizeUnifiedDiff(diff string) string {
	header := lipgloss.NewStyle().Foreground(termstyle.Faint)
	hunk := lipgloss.NewStyle().Foreground(termstyle.Purple)
	add := lipgloss.NewStyle().Foreground(termstyle.Green)
	del := lipgloss.NewStyle().Foreground(termstyle.Red)

	var b strings.Builder
	for _, line := range strings.Split(strings.TrimRight(diff, "\n"), "\n") {
		switch {
		case strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ "):
			b.WriteString(header.Render(line))
		case strings.HasPrefix(line, "@@ "):
			b.WriteString(hunk.Render(line))
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++ "):
			b.WriteString(add.Render(line))
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "--- "):
			b.WriteString(del.Render(line))
		default:
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func typesPresent(items []Item) map[string]int {
	m := make(map[string]int)
	for _, it := range items {
		m[it.ArtifactType]++
	}
	return m
}

func orderedTypes(present map[string]int) []string {
	order := []string{"adr", "spec", "guide", "arch", "open-item", "unknown"}
	var out []string
	for _, t := range order {
		if present[t] > 0 {
			out = append(out, t)
		}
	}
	// Include any unexpected types deterministically.
	var extra []string
	for t := range present {
		found := false
		for _, known := range order {
			if t == known {
				found = true
				break
			}
		}
		if !found {
			extra = append(extra, t)
		}
	}
	sort.Strings(extra)
	out = append(out, extra...)
	return out
}

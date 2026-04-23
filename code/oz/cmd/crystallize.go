package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/joaoajmatos/oz/internal/crystallize/classifier"
	"github.com/joaoajmatos/oz/internal/crystallize/classifier/heuristic"
	"github.com/joaoajmatos/oz/internal/crystallize/promote"
	"github.com/joaoajmatos/oz/internal/crystallize/review"
	"github.com/joaoajmatos/oz/internal/termstyle"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var crystallizeCmd = &cobra.Command{
	Use:   "crystallize",
	Short: "Report which notes likely belong in specs/, docs/, or ADRs",
	Long: `Classify notes under notes/ and report which ones likely belong in specs/, docs/, or ADRs.

Crystallize never writes files. Use --dry-run to print diffs for manual promotion.`,
	RunE: runCrystallize,
}

var (
	crDryRun   bool
	crTopic    string
	crNoEnrich bool
	crNoCache  bool
	crVerbose  bool
)

func init() {
	crystallizeCmd.Flags().BoolVar(&crDryRun, "dry-run", false, "print classifications and diffs without prompting")
	crystallizeCmd.Flags().StringVar(&crTopic, "topic", "", "filter notes to those containing this string (case-insensitive)")
	crystallizeCmd.Flags().BoolVar(&crNoEnrich, "no-enrich", false, "disable LLM classifier; use heuristic only")
	crystallizeCmd.Flags().BoolVar(&crNoCache, "no-cache", false, "disable cache reads; force fresh LLM calls")
	crystallizeCmd.Flags().BoolVar(&crVerbose, "verbose", false, "print verbose skip explanations")
}

type crystallizeItem struct {
	AbsPath string
	RelPath string
	Content []byte
	Class   classifier.Classification
}

type skippedItem struct {
	RelPath    string
	Type       string
	Confidence string
	Reason     string
	Content    []byte
}

func runCrystallize(cmd *cobra.Command, _ []string) error {
	root, err := findWorkspaceRoot()
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	in := cmd.InOrStdin()
	stderr := cmd.ErrOrStderr()

	fmt.Fprintln(out)
	fmt.Fprintf(out, "  %s  %s\n\n", termstyle.Brand.Render("oz"), termstyle.Subtle.Render("crystallize"))

	notesDir := filepath.Join(root, "notes")
	if _, err := os.Stat(notesDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("notes/ not found in workspace root")
		}
		return fmt.Errorf("stat notes/: %w", err)
	}

	paths, err := collectNotePaths(notesDir)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		fmt.Fprintln(out, termstyle.Subtle.Render("no notes to crystallize"))
		return nil
	}

	fmt.Fprintf(out, "%s %s\n\n", termstyle.Section.Render("Scan"), termstyle.Subtle.Render(fmt.Sprintf("(%d file(s))", len(paths))))

	progress := stderr
	showSpinner := isInteractiveWriter(stderr)
	// In report-only mode, keep output focused on the classification table even when stderr isn't a TTY.
	if !crDryRun && !showSpinner && !crVerbose {
		progress = io.Discard
	}

	items, err := classifyPaths(root, paths, crTopic, classifier.Options{
		WorkspaceRoot: root,
		NoEnrich:      crNoEnrich,
		NoCache:       crNoCache,
		Verbose: func(msg string) {
			if crVerbose {
				fmt.Fprintln(os.Stderr, msg)
			}
		},
	}, progress, showSpinner)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		fmt.Fprintln(out, termstyle.Subtle.Render("no notes matched (topic filter may have excluded all files)"))
		return nil
	}

	previewTargets := previewTargetPaths(root, items)
	fmt.Fprintln(out, termstyle.Section.Render("Classification"))
	printClassificationTable(out, items, previewTargets)

	// Default mode is report-only.
	if !crDryRun {
		fmt.Fprintln(out, "\nNext steps:")
		fmt.Fprintln(out, "  - Re-run with --dry-run to inspect proposed diffs.")
		fmt.Fprintln(out, "  - Manually promote (copy/move) notes into the target paths above.")
		return nil
	}

	// Separate unknowns (not promotable) from candidates.
	var candidates []crystallizeItem
	var skipped []skippedItem
	for _, it := range items {
		if it.Class.Type == classifier.TypeUnknown {
			skipped = append(skipped, skippedItem{
				RelPath:    it.RelPath,
				Type:       string(it.Class.Type),
				Confidence: string(it.Class.Confidence),
				Reason:     it.Class.Reason,
				Content:    it.Content,
			})
			continue
		}
		candidates = append(candidates, it)
	}

	if len(candidates) == 0 {
		fmt.Fprintln(out, "\n"+termstyle.Subtle.Render("no promotable notes (all classified as unknown)"))
		return nil
	}

	if len(skipped) > 0 && crVerbose {
		printVerboseSkips(out, skipped)
	}
	fmt.Fprintln(out, "\n"+termstyle.Section.Render("Dry Run"))
	if err := runCrystallizeDryRun(root, out, in, candidates, previewTargets); err != nil {
		return err
	}
	return nil
}

func collectNotePaths(notesDir string) ([]string, error) {
	var paths []string
	if err := filepath.WalkDir(notesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("walk notes/: %w", err)
	}
	sort.Strings(paths)
	return paths, nil
}

func classifyPaths(
	root string,
	paths []string,
	topic string,
	opts classifier.Options,
	progress io.Writer,
	showSpinner bool,
) ([]crystallizeItem, error) {
	c := classifier.New(opts)

	topic = strings.ToLower(strings.TrimSpace(topic))
	var out []crystallizeItem
	_, err := withSpinner(progress, showSpinner, "Classifying notes", func() (struct{}, error) {
		for i, p := range paths {
			rel := relFromRoot(root, p)
			if !showSpinner {
				fmt.Fprintf(progress, "%s %s\n", termstyle.Subtle.Render(fmt.Sprintf("[%d/%d]", i+1, len(paths))), rel)
			}
			content, err := os.ReadFile(p)
			if err != nil {
				return struct{}{}, fmt.Errorf("read %s: %w", p, err)
			}
			if topic != "" && !strings.Contains(strings.ToLower(string(content)), topic) {
				continue
			}

			cl, err := c.Classify(p)
			if err != nil {
				return struct{}{}, err
			}
			out = append(out, crystallizeItem{
				AbsPath: p,
				RelPath: rel,
				Content: content,
				Class:   cl,
			})
		}
		return struct{}{}, nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func isInteractiveWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd())
}

func withSpinner[T any](out io.Writer, enabled bool, label string, fn func() (T, error)) (T, error) {
	if !enabled {
		return fn()
	}

	done := make(chan struct{})
	defer close(done)

	var zero T
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	go func() {
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		i := 0
		for {
			select {
			case <-done:
				fmt.Fprintf(out, "\r%s %s\r", strings.Repeat(" ", len(label)+4), "")
				return
			case <-ticker.C:
				fmt.Fprintf(out, "\r%s %s", termstyle.Subtle.Render(frames[i%len(frames)]), termstyle.Subtle.Render(label))
				i++
			}
		}
	}()

	v, err := fn()
	if err != nil {
		return zero, err
	}
	fmt.Fprintf(out, "\r%s %s\n", termstyle.OK.Render("✓"), termstyle.Subtle.Render(label))
	return v, nil
}

func previewTargetPaths(root string, items []crystallizeItem) map[string]string {
	out := make(map[string]string, len(items))
	next, _ := promote.ADRNumber(filepath.Join(root, "specs", "decisions"))
	for _, it := range items {
		if it.Class.Type == classifier.TypeUnknown {
			continue
		}
		switch it.Class.Type {
		case classifier.TypeADR:
			out[it.RelPath] = relFromRoot(root, promote.ADRTargetPath(root, next, it.Class.Title))
			next++
		default:
			p, err := promote.TargetPath(root, string(it.Class.Type), it.Class.Title)
			if err != nil {
				out[it.RelPath] = "(error)"
			} else {
				out[it.RelPath] = relFromRoot(root, p)
			}
		}
	}
	return out
}

func runCrystallizeDryRun(root string, out io.Writer, in io.Reader, items []crystallizeItem, previewTargets map[string]string) error {
	now := time.Now()
	for i, it := range items {
		targetRel := previewTargets[it.RelPath]

		proposeOpts := promote.Options{Today: now}
		if it.Class.Type == classifier.TypeADR {
			// Use the preview ADR number inferred from the target path.
			base := filepath.Base(targetRel)
			if n, parseErr := parseADRPrefix(base); parseErr == nil {
				proposeOpts.ADRNumberOverride = n
			}
		}

		proposed, err := promote.ProposeContent(root, it.Content, string(it.Class.Type), it.Class.Title, proposeOpts)
		if err != nil {
			return err
		}

		_, _ = review.ReviewItem(review.Item{
			SourcePath:   it.RelPath,
			TargetPath:   targetRel,
			ArtifactType: string(it.Class.Type),
			Title:        it.Class.Title,
			Confidence:   string(it.Class.Confidence),
			Reason:       it.Class.Reason,
			Source:       it.Content,
			Proposed:     proposed,
		}, i+1, len(items), review.Options{Out: out, In: in, DryRun: true})
	}
	return nil
}

func parseADRPrefix(name string) (int, error) {
	if len(name) < 4 {
		return 0, fmt.Errorf("short name")
	}
	n := 0
	for i := 0; i < 4; i++ {
		if name[i] < '0' || name[i] > '9' {
			return 0, fmt.Errorf("not numeric")
		}
		n = n*10 + int(name[i]-'0')
	}
	return n, nil
}

func printClassificationTable(out io.Writer, items []crystallizeItem, previewTargets map[string]string) {
	const fileW = 44
	fmt.Fprintf(out, "  %-*s  %-9s  %-10s  %s\n", fileW, "FILE", "TYPE", "CONFIDENCE", "TARGET")
	fmt.Fprintln(out, "  "+strings.Repeat("─", fileW+41))
	for _, it := range items {
		name := it.RelPath
		if len(name) > fileW {
			name = "..." + name[len(name)-(fileW-3):]
		}
		target := previewTargets[it.RelPath]
		if target == "" {
			target = "—"
		}
		fmt.Fprintf(out, "  %-*s  %-9s  %-10s  %s\n", fileW, name, it.Class.Type, it.Class.Confidence, target)
	}
}

func printVerboseSkips(out io.Writer, skipped []skippedItem) {
	h := heuristic.New()
	fmt.Fprintln(out, "\nVerbose skip explanations:")
	for _, s := range skipped {
		r := h.Classify(s.RelPath, s.Content)
		t1, s1, t2, s2 := topTwoScores(r.Scores)
		fmt.Fprintf(out, "\n- %s\n  skip: %s (%s, %s)\n  candidates: %s=%d, %s=%d\n",
			s.RelPath, s.Reason, s.Type, s.Confidence, t1, s1, t2, s2)
	}
}

func topTwoScores(scores map[string]int) (t1 string, s1 int, t2 string, s2 int) {
	s1, s2 = -999, -999
	for t, s := range scores {
		switch {
		case s > s1:
			t2, s2 = t1, s1
			t1, s1 = t, s
		case s > s2:
			t2, s2 = t, s
		}
	}
	return t1, s1, t2, s2
}

func relFromRoot(root, abs string) string {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(rel)
}

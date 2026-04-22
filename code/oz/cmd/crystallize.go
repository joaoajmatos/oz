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

	"github.com/joaoajmatos/oz/internal/audit"
	"github.com/joaoajmatos/oz/internal/audit/coverage"
	"github.com/joaoajmatos/oz/internal/audit/orphans"
	"github.com/joaoajmatos/oz/internal/audit/staleness"
	ozcontext "github.com/joaoajmatos/oz/internal/context"
	"github.com/joaoajmatos/oz/internal/crystallize/classifier"
	"github.com/joaoajmatos/oz/internal/crystallize/classifier/heuristic"
	crlog "github.com/joaoajmatos/oz/internal/crystallize/log"
	"github.com/joaoajmatos/oz/internal/crystallize/promote"
	"github.com/joaoajmatos/oz/internal/crystallize/review"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var crystallizeCmd = &cobra.Command{
	Use:   "crystallize",
	Short: "Promote notes/ into canonical workspace artifacts",
	Long: `Classify notes under notes/ and promote them into canonical workspace artifacts.

Crystallize supports an interactive diff-first review flow, as well as non-interactive
automation via --dry-run and --accept-all.`,
	RunE: runCrystallize,
}

var (
	crDryRun    bool
	crTopic     string
	crAcceptAll bool
	crForce     bool
	crNoEnrich  bool
	crNoCache   bool
	crVerbose   bool
)

func init() {
	crystallizeCmd.Flags().BoolVar(&crDryRun, "dry-run", false, "skip all writes; print classifications and diffs without prompting")
	crystallizeCmd.Flags().StringVar(&crTopic, "topic", "", "filter notes to those containing this string (case-insensitive)")
	crystallizeCmd.Flags().BoolVar(&crAcceptAll, "accept-all", false, "auto-accept high-confidence results; skip interactive review")
	crystallizeCmd.Flags().BoolVar(&crForce, "force", false, "with --accept-all: also accept medium/low confidence results (never accepts unknown)")
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

type promotedItem struct {
	SourceRel string
	TargetRel string
	Type      string
	Appended  bool
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
	fmt.Fprintf(out, "  %s  %s\n\n", styleBrand.Render("oz"), styleSubtle.Render("crystallize"))

	if crForce && !crAcceptAll {
		fmt.Fprintln(os.Stderr, "warning: --force has no effect without --accept-all")
	}

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
		fmt.Fprintln(out, styleSubtle.Render("no notes to crystallize"))
		return nil
	}

	fmt.Fprintf(out, "%s %s\n\n", styleSectionTitle.Render("Scan"), styleSubtle.Render(fmt.Sprintf("(%d file(s))", len(paths))))

	items, err := classifyPaths(root, paths, crTopic, classifier.Options{
		WorkspaceRoot: root,
		NoEnrich:      crNoEnrich,
		NoCache:       crNoCache,
		Verbose: func(msg string) {
			if crVerbose {
				fmt.Fprintln(os.Stderr, msg)
			}
		},
	}, stderr, isInteractiveWriter(stderr))
	if err != nil {
		return err
	}

	if len(items) == 0 {
		fmt.Fprintln(out, styleSubtle.Render("no notes matched (topic filter may have excluded all files)"))
		return nil
	}

	previewTargets := previewTargetPaths(root, items)
	fmt.Fprintln(out, styleSectionTitle.Render("Classification"))
	printClassificationTable(out, items, previewTargets)

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

	if crDryRun {
		if len(skipped) > 0 && crVerbose {
			printVerboseSkips(out, skipped)
		}
		fmt.Fprintln(out, "\n"+styleSectionTitle.Render("Dry Run"))
		if err := runCrystallizeDryRun(root, out, in, candidates, previewTargets); err != nil {
			return err
		}
		return nil
	}

	if len(candidates) == 0 {
		fmt.Fprintln(out, "\n"+styleSubtle.Render("no promotable notes (all classified as unknown)"))
		return nil
	}

	// Batch mode for large backlogs (unless --accept-all is set).
	var batchPlan *review.BatchPlan
	if !crAcceptAll && len(candidates) > 15 {
		var summaryItems []review.Item
		for _, it := range items {
			summaryItems = append(summaryItems, review.Item{
				SourcePath:   it.RelPath,
				ArtifactType: string(it.Class.Type),
				Confidence:   string(it.Class.Confidence),
			})
		}
		fmt.Fprintln(out, "\n"+styleSectionTitle.Render("Batch Plan"))
		plan, err := review.BatchSummary(summaryItems, review.Options{Out: out, In: in, Theme: ozTheme()})
		if err != nil {
			return err
		}
		batchPlan = &plan
	}

	before, err := withSpinner(stderr, isInteractiveWriter(stderr), "Auditing (before)", func() (*audit.Report, error) {
		return runCrystallizeAudit(root)
	})
	if err != nil {
		return err
	}

	logger := crlog.New(filepath.Join(root, "context", "crystallize.log"))
	now := time.Now()

	var promoted []promotedItem
	var acceptAllSkipped []skippedItem

	total := len(candidates)
	for i := 0; i < len(candidates); i++ {
		it := candidates[i]

		// Apply batch plan (if any).
		if batchPlan != nil {
			switch batchPlan.ByType[string(it.Class.Type)] {
			case review.BatchSkip:
				skipped = append(skipped, skippedItem{
					RelPath:    it.RelPath,
					Type:       string(it.Class.Type),
					Confidence: string(it.Class.Confidence),
					Reason:     "skipped by batch plan",
					Content:    it.Content,
				})
				continue
			case review.BatchAccept:
				// accepted without prompting
			default:
				// review below
			}
		}

		// --accept-all path (non-interactive).
		if crAcceptAll {
			if it.Class.IsAutoAcceptable() || (crForce && it.Class.Type != classifier.TypeUnknown) {
				p, err := promote.Promote(root, it.RelPath, it.Content, string(it.Class.Type), it.Class.Title, promote.Options{Today: now})
				if err != nil {
					var col *promote.CollisionError
					if errors.As(err, &col) {
						skipped = append(skipped, skippedItem{
							RelPath:    it.RelPath,
							Type:       string(it.Class.Type),
							Confidence: string(it.Class.Confidence),
							Reason:     err.Error(),
							Content:    it.Content,
						})
						continue
					}
					return err
				}
				targetRel := relFromRoot(root, p.TargetPath)
				if rmErr := os.Remove(it.AbsPath); rmErr != nil {
					fmt.Fprintf(os.Stderr, "warning: remove %s: %v\n", it.RelPath, rmErr)
				}
				if err := logger.Append(it.RelPath, targetRel, string(it.Class.Type)); err != nil {
					return err
				}
				promoted = append(promoted, promotedItem{
					SourceRel: it.RelPath,
					TargetRel: targetRel,
					Type:      string(it.Class.Type),
					Appended:  p.Appended,
				})
				continue
			}

			acceptAllSkipped = append(acceptAllSkipped, skippedItem{
				RelPath:    it.RelPath,
				Type:       string(it.Class.Type),
				Confidence: string(it.Class.Confidence),
				Reason:     "not auto-acceptable",
				Content:    it.Content,
			})
			continue
		}

		// Interactive review (or batch plan "review").
		if batchPlan != nil && batchPlan.ByType[string(it.Class.Type)] == review.BatchAccept {
			p, err := promote.Promote(root, it.RelPath, it.Content, string(it.Class.Type), it.Class.Title, promote.Options{Today: now})
			if err != nil {
				var col *promote.CollisionError
				if errors.As(err, &col) {
					skipped = append(skipped, skippedItem{
						RelPath:    it.RelPath,
						Type:       string(it.Class.Type),
						Confidence: string(it.Class.Confidence),
						Reason:     err.Error(),
						Content:    it.Content,
					})
					continue
				}
				return err
			}
			targetRel := relFromRoot(root, p.TargetPath)
			if rmErr := os.Remove(it.AbsPath); rmErr != nil {
				fmt.Fprintf(os.Stderr, "warning: remove %s: %v\n", it.RelPath, rmErr)
			}
			if err := logger.Append(it.RelPath, targetRel, string(it.Class.Type)); err != nil {
				return err
			}
			promoted = append(promoted, promotedItem{
				SourceRel: it.RelPath,
				TargetRel: targetRel,
				Type:      string(it.Class.Type),
				Appended:  p.Appended,
			})
			continue
		}

		sourceContent := it.Content
		reviewedIdx := i + 1
		for {
			targetAbs, proposeOpts, err := previewTargetForItem(root, it.Class, now)
			if err != nil {
				return err
			}
			proposed, err := promote.ProposeContent(root, sourceContent, string(it.Class.Type), it.Class.Title, proposeOpts)
			if err != nil {
				return err
			}
			targetRel := relFromRoot(root, targetAbs)

			decision, err := review.ReviewItem(review.Item{
				SourcePath:   it.RelPath,
				TargetPath:   targetRel,
				ArtifactType: string(it.Class.Type),
				Title:        it.Class.Title,
				Confidence:   string(it.Class.Confidence),
				Reason:       it.Class.Reason,
				Source:       sourceContent,
				Proposed:     proposed,
			}, reviewedIdx, total, review.Options{Out: out, In: in, Theme: ozTheme()})
			if err != nil {
				return err
			}

			switch decision.Action {
			case review.ActionEdit:
				sourceContent = decision.Source
				continue
			case review.ActionSkip:
				skipped = append(skipped, skippedItem{
					RelPath:    it.RelPath,
					Type:       string(it.Class.Type),
					Confidence: string(it.Class.Confidence),
					Reason:     "skipped during review",
					Content:    sourceContent,
				})
			case review.ActionQuit:
				i = len(candidates) // exit outer loop
			case review.ActionAccept:
				p, err := promote.Promote(root, it.RelPath, sourceContent, string(it.Class.Type), it.Class.Title, promote.Options{Today: now})
				if err != nil {
					var col *promote.CollisionError
					if errors.As(err, &col) {
						skipped = append(skipped, skippedItem{
							RelPath:    it.RelPath,
							Type:       string(it.Class.Type),
							Confidence: string(it.Class.Confidence),
							Reason:     err.Error(),
							Content:    sourceContent,
						})
						break
					}
					return err
				}
				targetRelActual := relFromRoot(root, p.TargetPath)
				if rmErr := os.Remove(it.AbsPath); rmErr != nil {
					fmt.Fprintf(os.Stderr, "warning: remove %s: %v\n", it.RelPath, rmErr)
				}
				if err := logger.Append(it.RelPath, targetRelActual, string(it.Class.Type)); err != nil {
					return err
				}
				promoted = append(promoted, promotedItem{
					SourceRel: it.RelPath,
					TargetRel: targetRelActual,
					Type:      string(it.Class.Type),
					Appended:  p.Appended,
				})
			}
			break
		}
	}

	if crAcceptAll {
		printAcceptAllSummary(out, acceptAllSkipped)
		if crVerbose && len(acceptAllSkipped) > 0 {
			printVerboseSkips(out, acceptAllSkipped)
		}
	}

	if crVerbose && len(skipped) > 0 {
		printVerboseSkips(out, skipped)
	}

	fmt.Fprintf(out, "\nDone. %d file(s) promoted.\n", len(promoted))
	if len(promoted) > 0 {
		fmt.Fprintln(out)
		printReceipt(out, promoted)
		fmt.Fprintln(out)
		printUndoHint(out, promoted)
		fmt.Fprintln(out)
		fmt.Fprintln(out, styleSubtle.Render("Logged to context/crystallize.log"))
	}

	after, err := withSpinner(stderr, isInteractiveWriter(stderr), "Auditing (after)", func() (*audit.Report, error) {
		return runCrystallizeAudit(root)
	})
	if err != nil {
		return err
	}
	printAuditDelta(out, before, after)

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
				fmt.Fprintf(progress, "%s %s\n", styleSubtle.Render(fmt.Sprintf("[%d/%d]", i+1, len(paths))), rel)
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
				fmt.Fprintf(out, "\r%s %s", styleSubtle.Render(frames[i%len(frames)]), styleSubtle.Render(label))
				i++
			}
		}
	}()

	v, err := fn()
	if err != nil {
		return zero, err
	}
	fmt.Fprintf(out, "\r%s %s\n", styleSuccess.Render("✓"), styleSubtle.Render(label))
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

func previewTargetForItem(root string, cl classifier.Classification, today time.Time) (absTarget string, opts promote.Options, err error) {
	opts = promote.Options{Today: today}
	switch cl.Type {
	case classifier.TypeADR:
		n, err := promote.ADRNumber(filepath.Join(root, "specs", "decisions"))
		if err != nil {
			return "", promote.Options{}, err
		}
		opts.ADRNumberOverride = n
		return promote.ADRTargetPath(root, n, cl.Title), opts, nil
	default:
		p, err := promote.TargetPath(root, string(cl.Type), cl.Title)
		if err != nil {
			return "", promote.Options{}, err
		}
		return p, opts, nil
	}
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

func printAcceptAllSummary(out io.Writer, skipped []skippedItem) {
	if len(skipped) == 0 {
		return
	}
	fmt.Fprintf(out, "\nSkipped %d file(s) (not auto-acceptable):\n", len(skipped))
	for _, s := range skipped {
		fmt.Fprintf(out, "  - %s  (%s, %s)\n", s.RelPath, s.Type, s.Confidence)
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

func runCrystallizeAudit(root string) (*audit.Report, error) {
	// Refresh graph.json so audit checks reflect the current workspace.
	result, err := ozcontext.Build(root)
	if err != nil {
		return nil, fmt.Errorf("build context graph: %w", err)
	}
	if err := ozcontext.Serialize(root, result.Graph); err != nil {
		return nil, fmt.Errorf("write graph: %w", err)
	}

	checks := []audit.Check{
		&orphans.Check{},
		&coverage.Check{},
		&staleness.Check{},
	}
	return audit.RunAll(root, checks, audit.Options{})
}

func printAuditDelta(out io.Writer, before, after *audit.Report) {
	if before == nil || after == nil {
		return
	}
	be, bw, bi := before.Counts[audit.SeverityError], before.Counts[audit.SeverityWarn], before.Counts[audit.SeverityInfo]
	ae, aw, ai := after.Counts[audit.SeverityError], after.Counts[audit.SeverityWarn], after.Counts[audit.SeverityInfo]

	fmt.Fprintln(out, "\nAudit delta:")
	fmt.Fprintf(out, "  Before: %d error(s), %d warning(s), %d info\n", be, bw, bi)
	fmt.Fprintf(out, "  After:  %d error(s), %d warning(s), %d info  (%+d errors, %+d warnings, %+d info)\n",
		ae, aw, ai, ae-be, aw-bw, ai-bi)
}

func printReceipt(out io.Writer, items []promotedItem) {
	const srcW = 44
	fmt.Fprintf(out, "  %-*s  %-12s  %s\n", srcW, "SOURCE", "TYPE", "TARGET")
	fmt.Fprintln(out, "  "+strings.Repeat("─", srcW+34))
	for _, it := range items {
		src := it.SourceRel
		if len(src) > srcW {
			src = "..." + src[len(src)-(srcW-3):]
		}
		tgt := it.TargetRel
		if it.Appended {
			tgt += " (appended)"
		}
		fmt.Fprintf(out, "  %-*s  %-12s  %s\n", srcW, src, it.Type, tgt)
	}
}

func printUndoHint(out io.Writer, items []promotedItem) {
	var targets []string
	for _, it := range items {
		if it.TargetRel == "" {
			continue
		}
		targets = append(targets, it.TargetRel)
	}
	sort.Strings(targets)

	fmt.Fprintln(out, "To undo:")
	fmt.Fprintln(out, "  git checkout -- notes/")
	if len(targets) > 0 {
		fmt.Fprintf(out, "  git rm %s\n", strings.Join(targets, " "))
	}
}

func relFromRoot(root, abs string) string {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return filepath.ToSlash(abs)
	}
	return filepath.ToSlash(rel)
}

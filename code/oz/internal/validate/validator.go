package validate

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oz-tools/oz/internal/convention"
	"github.com/oz-tools/oz/internal/semantic"
	"github.com/oz-tools/oz/internal/workspace"
)

// Severity classifies a finding.
type Severity int

const (
	Error   Severity = iota // required item missing — fails validation
	Warning                 // recommended item missing — passes validation
)

// Finding is a single validation result.
type Finding struct {
	Severity Severity
	Message  string
}

// Result holds all findings from a validation run.
type Result struct {
	Findings []Finding
}

// Valid reports whether validation passed (no errors).
func (r *Result) Valid() bool {
	for _, f := range r.Findings {
		if f.Severity == Error {
			return false
		}
	}
	return true
}

func (r *Result) add(s Severity, format string, args ...any) {
	r.Findings = append(r.Findings, Finding{s, fmt.Sprintf(format, args...)})
}

// Validate runs all checks against the workspace at path.
func Validate(ws *workspace.Workspace) *Result {
	r := &Result{}
	checkRootFiles(ws, r)
	checkDirectories(ws, r)
	checkOZMD(ws, r)
	checkAgents(ws, r)
	checkSemanticOverlay(ws, r)
	return r
}

// checkSemanticOverlay warns when context/semantic.json contains unreviewed items.
// Missing overlay is not an error — it simply means 'oz context enrich' hasn't run yet.
func checkSemanticOverlay(ws *workspace.Workspace, r *Result) {
	o, err := semantic.Load(ws.Root)
	if err != nil || o == nil {
		return
	}
	unreviewed := 0
	for _, c := range o.Concepts {
		if !c.Reviewed {
			unreviewed++
		}
	}
	for _, e := range o.Edges {
		if !e.Reviewed {
			unreviewed++
		}
	}
	if unreviewed > 0 {
		r.add(Warning, "context/semantic.json has %d unreviewed item(s) — run 'oz context review'", unreviewed)
	}
}

func checkRootFiles(ws *workspace.Workspace, r *Result) {
	for file, status := range convention.RootFiles {
		path := filepath.Join(ws.Root, file)
		_, err := os.Stat(path)
		if err == nil {
			continue
		}
		switch status {
		case "required":
			r.add(Error, "missing required file: %s", file)
		case "recommended":
			r.add(Warning, "missing recommended file: %s", file)
		}
	}
}

func checkDirectories(ws *workspace.Workspace, r *Result) {
	for dir, status := range convention.Directories {
		path := filepath.Join(ws.Root, dir)
		_, err := os.Stat(path)
		if err == nil {
			continue
		}
		switch status {
		case "required":
			r.add(Error, "missing required directory: %s/", dir)
		case "recommended":
			r.add(Warning, "missing recommended directory: %s/", dir)
		}
	}
}

func checkOZMD(ws *workspace.Workspace, r *Result) {
	path := filepath.Join(ws.Root, "OZ.md")
	f, err := os.Open(path)
	if err != nil {
		// already caught by checkRootFiles
		return
	}
	defer f.Close()

	hasVersion := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "oz standard:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "oz standard:"))
			if v == "" {
				r.add(Error, "OZ.md: 'oz standard' field is empty")
			} else {
				hasVersion = true
			}
		}
	}
	if !hasVersion {
		r.add(Error, "OZ.md: missing 'oz standard' field")
	}
}

// requiredAgentSections are the H2 headings every AGENT.md must contain.
var requiredAgentSections = []string{
	"## Role",
	"## Read-chain",
	"## Rules",
	"## Skills",
	"## Responsibilities",
	"## Out of scope",
	"## Context topics",
}

func checkAgents(ws *workspace.Workspace, r *Result) {
	agentsDir := filepath.Join(ws.Root, "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		// agents/ absence already caught by checkDirectories
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		agentMD := filepath.Join(agentsDir, e.Name(), "AGENT.md")
		if _, err := os.Stat(agentMD); err != nil {
			r.add(Error, "agent %q: missing AGENT.md", e.Name())
			continue
		}
		checkAgentSections(agentMD, e.Name(), r)
	}
}

func checkAgentSections(path, name string, r *Result) {
	content, err := os.ReadFile(path)
	if err != nil {
		r.add(Error, "agent %q: could not read AGENT.md", name)
		return
	}
	body := string(content)
	for _, section := range requiredAgentSections {
		if !strings.Contains(body, section) {
			r.add(Error, "agent %q: AGENT.md missing section %q", name, section)
		}
	}
}

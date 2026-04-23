package cmd

import "math/rand/v2"

// ozTipz are one-line hints shown when you run `oz` with no subcommand.
// Keep them short; they rotate at random.
var ozTipz = []string{
	"After changing AGENTS.md, OZ.md, or agent files, run oz validate.",
	"oz context build refreshes context/graph.json from your tree.",
	"oz context query answers questions using the workspace graph.",
	"oz audit spots drift between specs, graph, and the repo layout.",
	"AGENTS.md routes models to the right agent — read the table first.",
	"Conventions in specs/ beat notes/; promote stable ideas out of notes/.",
	"oz add list shows optional skill and integration packages you can install.",
	"oz repair recreates missing default files without overwriting yours.",
	"oz context serve exposes the graph to tools over MCP for editors.",
	"oz crystallize helps turn raw notes into specs, ADRs, or docs — explore the subcommands.",
	"Code is the source of truth for behavior; the spec should follow when they diverge.",
	"Run oz context build after meaningful edits to .md or code so the graph stays fresh.",
}

func randomOzTip() string {
	if len(ozTipz) == 0 {
		return ""
	}
	return ozTipz[rand.IntN(len(ozTipz))]
}

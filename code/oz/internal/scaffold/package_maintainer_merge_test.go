package scaffold

import (
	"strings"
	"testing"
)

func TestMergeAgentsMaintainer_ScaffoldStyle(t *testing.T) {
	in := `## Agents

| Agent | Use when | Definition |
|---|---|---|
| **coding** | Builds code | ` + "`agents/coding/AGENT.md`" + ` |

## Source of Truth Hierarchy

x`
	out, changed, err := mergeAgentsMaintainer(in)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected merge")
	}
	if !strings.Contains(out, "| **maintainer** |") {
		t.Fatal(out)
	}
}

func TestMergeAgentsMaintainer_Idempotent(t *testing.T) {
	in := `## Agents

| Agent | Use when | Definition |
|---|---|---|
| **maintainer** | Already | ` + "`agents/maintainer/AGENT.md`" + ` |

## Source of Truth Hierarchy
`
	out, changed, err := mergeAgentsMaintainer(in)
	if err != nil {
		t.Fatal(err)
	}
	if changed || out != in {
		t.Fatalf("expected no change, changed=%v", changed)
	}
}

func TestMergeOZMaintainer_Table(t *testing.T) {
	in := `## Registered Agents

| Agent | Use when | Definition |
|---|---|---|
| **coding** | r | ` + "`agents/coding/AGENT.md`" + ` |

## Source of Truth Hierarchy
`
	out, changed, err := mergeOZMaintainer(in)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected merge")
	}
	if !strings.Contains(out, "| **maintainer** |") {
		t.Fatal(out)
	}
}

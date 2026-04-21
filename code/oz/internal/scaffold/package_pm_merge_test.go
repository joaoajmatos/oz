package scaffold

import (
	"strings"
	"testing"
)

func TestMergeAgentsPM_WithSeparatorBeforeSource(t *testing.T) {
	in := `# AGENTS

## Agents

| Agent | Use when | Definition |
|---|---|---|
| **other** | Does things | ` + "`agents/other/AGENT.md`" + ` |

---

## Source of Truth Hierarchy

1. specs
`
	out, changed, err := mergeAgentsPM(in)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected merge")
	}
	if !strings.Contains(out, "| **pm** |") || !strings.Contains(out, "agents/pm/AGENT.md") {
		t.Fatalf("missing pm row:\n%s", out)
	}
	if !strings.Contains(out, "## Source of Truth Hierarchy") {
		t.Fatal("lost Source heading")
	}
}

func TestMergeAgentsPM_ScaffoldStyle(t *testing.T) {
	in := `## Agents

| Agent | Use when | Definition |
|---|---|---|
| **coding** | Builds code | ` + "`agents/coding/AGENT.md`" + ` |

## Source of Truth Hierarchy

x`
	out, changed, err := mergeAgentsPM(in)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected merge")
	}
	if !strings.Contains(out, "| **pm** |") {
		t.Fatal(out)
	}
}

func TestMergeAgentsPM_Idempotent(t *testing.T) {
	in := `## Agents

| Agent | Use when | Definition |
|---|---|---|
| **pm** | Already | ` + "`agents/pm/AGENT.md`" + ` |

## Source of Truth Hierarchy
`
	out, changed, err := mergeAgentsPM(in)
	if err != nil {
		t.Fatal(err)
	}
	if changed || out != in {
		t.Fatalf("expected no change, changed=%v", changed)
	}
}

func TestMergeOZPM_Table(t *testing.T) {
	in := `## Registered Agents

| Agent | Use when | Definition |
|---|---|---|
| **a** | r | ` + "`agents/a/AGENT.md`" + ` |

## Source of Truth Hierarchy
`
	out, changed, err := mergeOZPM(in)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected merge")
	}
	if !strings.Contains(out, "| **pm** |") {
		t.Fatal(out)
	}
}

func TestMergeOZPM_List(t *testing.T) {
	in := `## Registered Agents

- **coding**: ` + "`agents/coding/AGENT.md`" + `

## Source of Truth Hierarchy
`
	out, changed, err := mergeOZPM(in)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected merge")
	}
	if !strings.Contains(out, "- **pm**:") {
		t.Fatal(out)
	}
}

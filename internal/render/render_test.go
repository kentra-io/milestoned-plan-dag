package render

import (
	"os"
	"strings"
	"testing"

	"github.com/kentra-io/milestoned-plan-dag/internal/plan"
)

func loadFixture(t *testing.T, path string) *plan.Plan {
	t.Helper()
	p, err := plan.LoadFile(path)
	if err != nil {
		t.Fatalf("load fixture %s: %v", path, err)
	}
	return p
}

// TestRender_ShapeAndCheckboxes covers "render emits a read-only human
// markdown view (checkbox steps + contract block) from the same model": a
// heading per milestone, Goal, Deliverables, a contract block, and steps
// rendered as - [ ] / - [x].
func TestRender_ShapeAndCheckboxes(t *testing.T) {
	p := loadFixture(t, "../../testdata/valid/branching.yaml")

	out, err := Render(p)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	if !strings.Contains(out, "## Milestone 1: Foundation") {
		t.Fatalf("missing milestone 1 heading, got:\n%s", out)
	}
	if !strings.Contains(out, "**Goal** — Foundation") {
		t.Fatalf("missing Goal line, got:\n%s", out)
	}
	if !strings.Contains(out, "**Deliverables** — The shared foundation both branches build on.") {
		t.Fatalf("missing Deliverables line, got:\n%s", out)
	}
	if !strings.Contains(out, "- [x] lay the foundation") {
		t.Fatalf("missing done checkbox, got:\n%s", out)
	}
	if !strings.Contains(out, "- [ ] build branch b") {
		t.Fatalf("missing pending checkbox, got:\n%s", out)
	}
	if !strings.Contains(out, "check: none") {
		t.Fatalf("missing contract check line, got:\n%s", out)
	}
}

// TestRender_Golden pins the exact markdown shape against a committed
// fixture.
func TestRender_Golden(t *testing.T) {
	p := loadFixture(t, "../../testdata/valid/branching.yaml")

	got, err := Render(p)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	want, err := os.ReadFile("../../testdata/golden/branching.render.md")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if got != string(want) {
		t.Fatalf("render output does not match golden:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// TestRender_StructuredSlots covers the structured deliverables and
// test-case-shaped criteria optional slots rendering into readable markdown.
func TestRender_StructuredSlots(t *testing.T) {
	p := loadFixture(t, "../../testdata/valid/optional-slots-present.yaml")

	got, err := Render(p)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	want, err := os.ReadFile("../../testdata/golden/optional-slots-present.render.md")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if got != string(want) {
		t.Fatalf("render output does not match golden:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

// The plan preamble, milestone description and rules render, and a
// multi-line step stays one list item: continuation lines are indented two
// spaces so a fenced code block inside a step does not break the list.
func TestRender_ContextRulesAndMultilineSteps(t *testing.T) {
	p := loadFixture(t, "../../testdata/valid/context-and-rules.yaml")

	out, err := Render(p)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	for _, want := range []string{
		"## Plan\n\nAdds a greeting endpoint. Every milestone follows the hexagonal layout.\n\n**Rules** — java-hexagonal, constitution/constitution.md\n\n## Milestone 1: Greeter domain service",
		"**Goal** — Greeter domain service\n\n**Description**\n\nThe domain service the adapter in milestone 2 calls.\n\n**Rules** — docs/naming.md\n\n**Deliverables**",
		"- [ ] Write the failing test\n  ```java\n  assertThat(new Greeter().greet(\"Ada\")).isEqualTo(\"Hello, Ada\");\n  ```\n- [ ] Implement Greeter.greet\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render output missing:\n%s\n--- got:\n%s", want, out)
		}
	}
}

// TestRender_ContextGolden pins the exact markdown for the new fields.
func TestRender_ContextGolden(t *testing.T) {
	p := loadFixture(t, "../../testdata/valid/context-and-rules.yaml")
	got, err := Render(p)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	want, err := os.ReadFile("../../testdata/golden/context-and-rules.render.md")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if got != string(want) {
		t.Fatalf("render output does not match golden:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

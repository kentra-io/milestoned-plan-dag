// Package render projects a plan into a read-only human markdown view,
// mirroring the authoring tasks.md shape (plan-projection: "render emits a
// read-only human markdown view (Goal / Deliverables / Validation contract /
// checkbox steps) from the same model"). Nothing in this primitive ever
// parses this markdown back — it is a one-way projection generated straight
// from the typed plan model, never from the resolve YAML output.
package render

import (
	"fmt"
	"strings"

	"github.com/kentra-io/milestoned-plan-dag/internal/plan"
)

// Render produces the full markdown document for a plan: one "## Milestone
// N: <goal>" section per milestone, in document order, each carrying its
// Goal, Deliverables, a fenced contract block, and its steps as `- [ ]`/`-
// [x]` checkboxes.
func Render(p *plan.Plan) (string, error) {
	var b strings.Builder
	for i, m := range p.Milestones {
		if i > 0 {
			b.WriteString("\n")
		}
		renderMilestone(&b, m)
	}
	return b.String(), nil
}

// renderMilestone appends one milestone's markdown section to b.
func renderMilestone(b *strings.Builder, m plan.Milestone) {
	fmt.Fprintf(b, "## Milestone %d: %s\n\n", m.Number, m.Goal)
	fmt.Fprintf(b, "**Goal** — %s\n\n", m.Goal)
	fmt.Fprintf(b, "**Deliverables** — %s\n\n", renderDeliverables(m.Deliverables))

	b.WriteString("**Validation contract**\n\n")
	b.WriteString("```contract\n")
	renderContract(b, m.Contract)
	b.WriteString("```\n\n")

	b.WriteString("**Steps**\n\n")
	renderSteps(b, m.Steps)
}

// renderDeliverables renders the prose-or-structured deliverables slot as a
// single readable line: prose verbatim, or a create/modify/test summary.
func renderDeliverables(d plan.Deliverables) string {
	if d.Prose != "" {
		return d.Prose
	}
	if !d.Structured() {
		return ""
	}
	var parts []string
	if len(d.Create) > 0 {
		parts = append(parts, "create: "+strings.Join(d.Create, ", "))
	}
	if len(d.Modify) > 0 {
		parts = append(parts, "modify: "+strings.Join(d.Modify, ", "))
	}
	if len(d.Test) > 0 {
		parts = append(parts, "test: "+strings.Join(d.Test, ", "))
	}
	return strings.Join(parts, "; ")
}

// renderContract writes the check/criteria/paths lines of a milestone's
// contract into the fenced block b is currently writing.
func renderContract(b *strings.Builder, c plan.Contract) {
	fmt.Fprintf(b, "check: %s\n", c.Check)
	renderCriteria(b, c.Criteria)
	renderPathsList(b, "paths", c.Paths)
}

// renderCriteria writes the criteria key: plain text on one line, or a
// case-shaped list, one bullet per test case.
func renderCriteria(b *strings.Builder, c plan.Criteria) {
	if len(c.Cases) == 0 {
		fmt.Fprintf(b, "criteria: %s\n", c.Text)
		return
	}
	b.WriteString("criteria:\n")
	for _, tc := range c.Cases {
		fmt.Fprintf(b, "  - %s: given %s, when %s, then %s\n", tc.Name, tc.Given, tc.When, tc.Then)
	}
}

// renderPathsList writes a `key:` line followed by one `  - <path>` bullet
// per entry, or `key: []` when the list is empty.
func renderPathsList(b *strings.Builder, key string, paths []string) {
	if len(paths) == 0 {
		fmt.Fprintf(b, "%s: []\n", key)
		return
	}
	fmt.Fprintf(b, "%s:\n", key)
	for _, path := range paths {
		fmt.Fprintf(b, "  - %s\n", path)
	}
}

// renderSteps writes one `- [ ]`/`- [x]` bullet per step.
func renderSteps(b *strings.Builder, steps []plan.Step) {
	for _, s := range steps {
		mark := " "
		if s.Done {
			mark = "x"
		}
		fmt.Fprintf(b, "- [%s] %s\n", mark, s.Text)
	}
}

package dag

import (
	"testing"
)

// plan-projection "Independent milestones are linearized deterministically":
// a milestone appears after every milestone it depends on, and ready
// milestones are emitted in ascending `number` order.
func TestTopoOrderBranchingIsDeterministic(t *testing.T) {
	g := Build(load(t, "branching.yaml"))
	got, err := g.TopoOrder()
	if err != nil {
		t.Fatalf("TopoOrder: %v", err)
	}
	want := []string{"a", "b", "c"} // b (number 2) before c (number 3), both after a
	if !equal(got, want) {
		t.Fatalf("TopoOrder = %v, want %v", got, want)
	}
}

// The tie-break is milestone `number`, not document order: with 2 and 3 both
// depending only on 1, 2 must precede 3 even if 3 is authored first.
func TestTopoOrderTieBreaksByNumberNotDocumentOrder(t *testing.T) {
	g := Build(loadInline(t, `
schemaVersion: 0.1.0
milestones:
  - number: 1
    slug: root
    goal: Root
    contract: {check: none, criteria: c, paths: []}
    steps: ["[ ] x"]
  - number: 3
    slug: later
    goal: Later
    contract: {check: none, criteria: c, paths: []}
    depends-on: [root]
    steps: ["[ ] x"]
  - number: 2
    slug: earlier
    goal: Earlier
    contract: {check: none, criteria: c, paths: []}
    depends-on: [root]
    steps: ["[ ] x"]
`))
	got, err := g.TopoOrder()
	if err != nil {
		t.Fatalf("TopoOrder: %v", err)
	}
	want := []string{"root", "earlier", "later"} // earlier=number 2 before later=number 3
	if !equal(got, want) {
		t.Fatalf("TopoOrder = %v, want %v", got, want)
	}
}

func TestTopoOrderRejectsACycle(t *testing.T) {
	g := Build(loadInline(t, `
schemaVersion: 0.1.0
milestones:
  - number: 1
    slug: x
    goal: X
    contract: {check: none, criteria: c, paths: []}
    depends-on: [y]
    steps: ["[ ] x"]
  - number: 2
    slug: y
    goal: Y
    contract: {check: none, criteria: c, paths: []}
    depends-on: [x]
    steps: ["[ ] x"]
`))
	if _, err := g.TopoOrder(); err == nil {
		t.Fatal("TopoOrder should error on a cycle, got nil")
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

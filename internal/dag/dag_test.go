package dag

import (
	"path/filepath"
	"testing"

	"milestoned-plan-dag/internal/plan"
)

const validDir = "../../testdata/valid"

func load(t *testing.T, name string) *plan.Plan {
	t.Helper()
	p, err := plan.LoadFile(filepath.Join(validDir, name))
	if err != nil {
		t.Fatalf("LoadFile(%s): %v", name, err)
	}
	return p
}

func loadInline(t *testing.T, y string) *plan.Plan {
	t.Helper()
	p, err := plan.Load([]byte(y))
	if err != nil {
		t.Fatalf("Load inline: %v", err)
	}
	return p
}

func deps(g *Graph, slug string) []string { return g.Deps[slug] }

// plan-schema "No explicit edges yields a sequential chain": a plan with no
// depends-on resolves to an implicit preceding-milestone chain.
func TestImplicitChain(t *testing.T) {
	g := Build(load(t, "sequential.yaml"))
	if got := deps(g, "scaffold"); len(got) != 0 {
		t.Errorf("first milestone should have no edge, got %v", got)
	}
	if got := deps(g, "model"); len(got) != 1 || got[0] != "scaffold" {
		t.Errorf("model deps = %v, want [scaffold]", got)
	}
	if got := deps(g, "validate"); len(got) != 1 || got[0] != "model" {
		t.Errorf("validate deps = %v, want [model]", got)
	}
	if !g.Reachable("validate", "scaffold") {
		t.Errorf("validate should transitively depend on scaffold")
	}
}

// plan-schema "Explicit edges yield a branching DAG": b and c both depend on a
// and neither depends on the other.
func TestExplicitBranching(t *testing.T) {
	g := Build(load(t, "branching.yaml"))
	if got := deps(g, "b"); len(got) != 1 || got[0] != "a" {
		t.Errorf("b deps = %v, want [a]", got)
	}
	if got := deps(g, "c"); len(got) != 1 || got[0] != "a" {
		t.Errorf("c deps = %v, want [a]", got)
	}
	if !g.Independent("b", "c") {
		t.Errorf("b and c should be independent")
	}
	if g.Reachable("b", "c") || g.Reachable("c", "b") {
		t.Errorf("neither branch should reach the other")
	}
	if !g.Reachable("b", "a") {
		t.Errorf("b should reach a")
	}
}

// plan-validation "A valid plan is an acyclic dependency graph": a valid DAG
// reports no cycle.
func TestNoCycleForDAG(t *testing.T) {
	if c := Build(load(t, "branching.yaml")).Cycle(); c != nil {
		t.Errorf("branching plan should be acyclic, got cycle %v", c)
	}
}

func TestCycleDetectedAndNamed(t *testing.T) {
	g := Build(loadInline(t, `schemaVersion: 0.1.0
milestones:
  - number: 1
    slug: a
    goal: a
    contract: {check: none, criteria: x, paths: []}
    depends-on: [b]
    steps: ["[ ] s"]
  - number: 2
    slug: b
    goal: b
    contract: {check: none, criteria: x, paths: []}
    depends-on: [a]
    steps: ["[ ] s"]
`))
	c := g.Cycle()
	if len(c) != 2 {
		t.Fatalf("cycle = %v, want 2 members", c)
	}
	has := map[string]bool{}
	for _, s := range c {
		has[s] = true
	}
	if !has["a"] || !has["b"] {
		t.Errorf("cycle %v should name a and b", c)
	}
}

func TestReachableNotSelfForAcyclic(t *testing.T) {
	g := Build(load(t, "branching.yaml"))
	if g.Reachable("a", "a") {
		t.Errorf("a should not be reachable from itself in an acyclic graph")
	}
}

// plan-schema "A milestone is addressable by a position-independent slug":
// a depends-on edge resolves by slug regardless of the target's document
// position / number.
func TestSlugPositionIndependent(t *testing.T) {
	early := Build(loadInline(t, `schemaVersion: 0.1.0
milestones:
  - number: 1
    slug: base
    goal: base
    contract: {check: none, criteria: x, paths: []}
    steps: ["[ ] s"]
  - number: 2
    slug: parse-dag
    goal: parse
    contract: {check: none, criteria: x, paths: []}
    depends-on: [base]
    steps: ["[ ] s"]
  - number: 3
    slug: consumer
    goal: consume
    contract: {check: none, criteria: x, paths: []}
    depends-on: [parse-dag]
    steps: ["[ ] s"]
`))
	// parse-dag moved earlier (now number 1, first in document); consumer's
	// depends-on: [parse-dag] must still resolve.
	moved := Build(loadInline(t, `schemaVersion: 0.1.0
milestones:
  - number: 1
    slug: parse-dag
    goal: parse
    contract: {check: none, criteria: x, paths: []}
    steps: ["[ ] s"]
  - number: 2
    slug: base
    goal: base
    contract: {check: none, criteria: x, paths: []}
    depends-on: [parse-dag]
    steps: ["[ ] s"]
  - number: 3
    slug: consumer
    goal: consume
    contract: {check: none, criteria: x, paths: []}
    depends-on: [parse-dag]
    steps: ["[ ] s"]
`))
	for _, g := range []*Graph{early, moved} {
		if !g.Defined("parse-dag") {
			t.Errorf("parse-dag should be defined")
		}
		if !g.Reachable("consumer", "parse-dag") {
			t.Errorf("consumer's depends-on [parse-dag] should resolve regardless of position")
		}
	}
}

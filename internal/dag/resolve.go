// Package dag resolves a plan's milestone dependency relation into a directed
// graph and answers the structural questions the semantic validator needs:
// which edges exist, whether they form a cycle, and whether a dependency path
// runs between two milestones (reachability).
//
// Edge model (plan-schema "Dependency edges form a DAG"): a milestone depends
// on exactly the slugs its `depends-on` lists. A milestone that lists none is
// a root and can start at once; no edge is ever inferred from document order.
// Because edges reference slugs, they are position-independent: reordering
// milestones (changing their numbers) does not change any edge.
package dag

import "github.com/kentra-io/milestoned-plan-dag/internal/plan"

// Graph is a resolved milestone dependency graph. An edge "s depends on d"
// (d must complete before s) is recorded as d ∈ Deps[s].
type Graph struct {
	// Slugs lists milestone slugs in document order.
	Slugs []string
	// Deps maps a milestone slug to the slugs it depends on — exactly its declared depends-on.
	Deps map[string][]string
	// Numbers maps a milestone slug to its ordinal `number`, used as the
	// deterministic tie-break when linearizing independent milestones.
	Numbers map[string]int
	// defined is the set of slugs a milestone in the plan actually declares,
	// used to ignore dangling references during traversal.
	defined map[string]struct{}
}

// Build resolves the plan's edge set: each milestone's explicit `depends-on`,
// nothing else. Slug identity is used verbatim; duplicate slugs (rejected
// elsewhere by the validator) collapse in the map and are not Build's concern.
func Build(p *plan.Plan) *Graph {
	g := &Graph{
		Deps:    make(map[string][]string, len(p.Milestones)),
		Numbers: make(map[string]int, len(p.Milestones)),
		defined: make(map[string]struct{}, len(p.Milestones)),
	}
	for _, m := range p.Milestones {
		g.Slugs = append(g.Slugs, m.Slug)
		g.defined[m.Slug] = struct{}{}
		g.Numbers[m.Slug] = m.Number
	}
	for _, m := range p.Milestones {
		g.Deps[m.Slug] = append([]string(nil), m.DependsOn...)
	}
	return g
}

// Defined reports whether slug is declared by a milestone in the plan.
func (g *Graph) Defined(slug string) bool {
	_, ok := g.defined[slug]
	return ok
}

// Reachable reports whether there is a dependency path from `from` to `to` —
// i.e. whether `from` (transitively) depends on `to`. A milestone is not
// considered reachable from itself unless it participates in a cycle.
// Dangling references (slugs not defined in the plan) are skipped.
func (g *Graph) Reachable(from, to string) bool {
	seen := make(map[string]struct{}, len(g.Slugs))
	var walk func(string) bool
	walk = func(n string) bool {
		for _, dep := range g.Deps[n] {
			if !g.Defined(dep) {
				continue
			}
			if dep == to {
				return true
			}
			if _, ok := seen[dep]; ok {
				continue
			}
			seen[dep] = struct{}{}
			if walk(dep) {
				return true
			}
		}
		return false
	}
	return walk(from)
}

// Independent reports whether two distinct milestones have no dependency path
// between them in either direction, and therefore could be runnable
// concurrently by a future executor. A milestone is never independent of
// itself.
func (g *Graph) Independent(a, b string) bool {
	if a == b {
		return false
	}
	return !g.Reachable(a, b) && !g.Reachable(b, a)
}

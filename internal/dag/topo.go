package dag

import (
	"fmt"
	"sort"
)

// TopoOrder returns a deterministic topological ordering of the plan's
// milestones (by slug): a milestone appears only after every milestone it
// depends on, and whenever several milestones become ready together, the one
// with the smaller `number` is emitted first (plan-projection "Independent
// milestones are linearized deterministically"). The number tie-break makes
// the order reproducible and independent of document order.
//
// It returns an error when the graph contains a cycle — the remaining
// milestones can never become ready, so no valid order exists. (The semantic
// validator rejects cycles earlier; TopoOrder stays self-contained so a
// consumer that skips validation still fails loudly rather than looping.)
func (g *Graph) TopoOrder() ([]string, error) {
	// Unmet dependency count per slug, over edges to milestones that actually
	// exist (dangling references are ignored, matching Reachable).
	remaining := make(map[string]int, len(g.Slugs))
	for _, s := range g.Slugs {
		for _, d := range g.Deps[s] {
			if g.Defined(d) {
				remaining[s]++
			}
		}
	}

	emitted := make(map[string]struct{}, len(g.Slugs))
	order := make([]string, 0, len(g.Slugs))
	for len(order) < len(g.Slugs) {
		// Every slug whose dependencies are all already emitted.
		var ready []string
		for _, s := range g.Slugs {
			if _, done := emitted[s]; done {
				continue
			}
			if remaining[s] == 0 {
				ready = append(ready, s)
			}
		}
		if len(ready) == 0 {
			return nil, fmt.Errorf("plan has a dependency cycle: cannot order %d remaining milestone(s)", len(g.Slugs)-len(order))
		}
		// Deterministic pick: smallest number, then slug as a final tie-break.
		sort.Slice(ready, func(i, j int) bool {
			if g.Numbers[ready[i]] != g.Numbers[ready[j]] {
				return g.Numbers[ready[i]] < g.Numbers[ready[j]]
			}
			return ready[i] < ready[j]
		})
		next := ready[0]
		order = append(order, next)
		emitted[next] = struct{}{}
		// Releasing `next` decrements its dependents' unmet counts.
		for _, s := range g.Slugs {
			if _, done := emitted[s]; done {
				continue
			}
			for _, d := range g.Deps[s] {
				if d == next {
					remaining[s]--
				}
			}
		}
	}
	return order, nil
}

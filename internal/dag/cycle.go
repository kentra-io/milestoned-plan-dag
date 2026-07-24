package dag

// Cycle returns the members of a dependency cycle in document order of
// discovery, or nil when the graph is acyclic. A valid plan is acyclic
// (plan-validation "A valid plan is an acyclic dependency graph"), so a
// non-nil return names the offending milestones for the validator to report.
//
// Only defined slugs are traversed; dangling references are skipped so a
// separate dangling-edge check owns that failure. The returned slice lists the
// cycle from the first re-entered node around back to it (e.g. [a b] for
// a→b→a).
func (g *Graph) Cycle() []string {
	const (
		white = 0 // unvisited
		gray  = 1 // on the current DFS stack
		black = 2 // fully explored
	)
	color := make(map[string]int, len(g.Slugs))
	var stack []string

	var dfs func(string) []string
	dfs = func(n string) []string {
		color[n] = gray
		stack = append(stack, n)
		for _, dep := range g.Deps[n] {
			if !g.Defined(dep) {
				continue
			}
			switch color[dep] {
			case gray:
				// Re-entered a node still on the stack: the cycle is the
				// stack slice from that node to the current tip.
				for i, s := range stack {
					if s == dep {
						return append([]string(nil), stack[i:]...)
					}
				}
			case white:
				if c := dfs(dep); c != nil {
					return c
				}
			}
		}
		stack = stack[:len(stack)-1]
		color[n] = black
		return nil
	}

	for _, s := range g.Slugs {
		if color[s] == white {
			if c := dfs(s); c != nil {
				return c
			}
		}
	}
	return nil
}

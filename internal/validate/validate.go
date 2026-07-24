// Package validate is the authoritative semantic validator for a plan. It runs
// JSON-Schema shape validation first, then the semantic DAG guarantees a JSON
// Schema cannot express: identity uniqueness, edge resolution, acyclicity, and
// contract well-formedness (all rejections, naming the offender), plus the
// non-fatal overlapping-write-paths warning for independent milestones.
package validate

import (
	"fmt"
	"sort"
	"strings"

	"milestoned-plan-dag/internal/dag"
	"milestoned-plan-dag/internal/plan"
	"milestoned-plan-dag/internal/schema"
)

// Result is the outcome of validating a plan: fatal Errors (a non-empty slice
// means the plan is rejected) and non-fatal Warnings (hazards that still pass).
type Result struct {
	Errors   []string
	Warnings []string
}

// OK reports whether the plan is accepted (no fatal errors). Warnings do not
// affect acceptance.
func (r *Result) OK() bool { return len(r.Errors) == 0 }

// Validate shape-validates then semantically validates a raw plan YAML
// document. It returns a Result carrying every fatal error and every warning;
// the second return is non-nil only when the document cannot be parsed at all
// (so no semantic pass is possible).
func Validate(planYAML []byte) (*Result, error) {
	r := &Result{}

	// Tier 1: shape (JSON Schema). Collected, not short-circuited, so the
	// semantic pass can still name offenders precisely.
	if err := schema.Validate(planYAML); err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("shape: %v", err))
	}

	// Tier 2: semantic. Requires a parsed model.
	p, err := plan.Load(planYAML)
	if err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("parse: %v", err))
		return r, err
	}

	checkIdentity(p, r)
	checkContracts(p, r)
	dangling := checkDependencies(p, r)

	// Cycle detection and path-overlap analysis need a well-formed edge set:
	// run them only when identity is unique and no edge dangles.
	if !dangling && !hasDuplicateSlug(p) {
		g := dag.Build(p)
		if cycle := g.Cycle(); cycle != nil {
			r.Errors = append(r.Errors, fmt.Sprintf(
				"cycle: milestones %s form a dependency cycle", strings.Join(cycle, " -> ")))
		} else {
			warnOverlappingPaths(p, g, r)
		}
	}

	return r, nil
}

// checkIdentity rejects duplicate slugs and duplicate numbers (plan-validation
// "Milestone identity is unique").
func checkIdentity(p *plan.Plan, r *Result) {
	slugSeen := map[string]bool{}
	numSeen := map[int]bool{}
	for _, m := range p.Milestones {
		if slugSeen[m.Slug] {
			r.Errors = append(r.Errors, fmt.Sprintf("duplicate slug: %q is declared by more than one milestone", m.Slug))
		}
		slugSeen[m.Slug] = true
		if numSeen[m.Number] {
			r.Errors = append(r.Errors, fmt.Sprintf("duplicate number: %d is declared by more than one milestone", m.Number))
		}
		numSeen[m.Number] = true
	}
}

// hasDuplicateSlug reports whether any slug repeats (edges become ambiguous).
func hasDuplicateSlug(p *plan.Plan) bool {
	seen := map[string]bool{}
	for _, m := range p.Milestones {
		if seen[m.Slug] {
			return true
		}
		seen[m.Slug] = true
	}
	return false
}

// checkContracts rejects a milestone whose contract is missing or malformed:
// an empty check (neither a command nor the `none` sentinel), an empty
// criteria, or an absent paths key (plan-validation "Every milestone has a
// well-formed contract"). An empty paths list (`paths: []`) is well-formed.
func checkContracts(p *plan.Plan, r *Result) {
	for _, m := range p.Milestones {
		c := m.Contract
		var problems []string
		if strings.TrimSpace(c.Check) == "" {
			problems = append(problems, "check is empty (expected a command or the `none` sentinel)")
		}
		if c.Criteria.Text == "" && len(c.Criteria.Cases) == 0 {
			problems = append(problems, "criteria is empty")
		}
		if c.Paths == nil {
			problems = append(problems, "paths key is absent (use `paths: []` for an empty write-set)")
		}
		if len(problems) > 0 {
			r.Errors = append(r.Errors, fmt.Sprintf(
				"milestone %q (number %d) has a missing or malformed contract: %s",
				m.Slug, m.Number, strings.Join(problems, "; ")))
		}
	}
}

// checkDependencies rejects depends-on references to undefined slugs
// (plan-validation "Dependency edges resolve to existing milestones"). It
// returns whether any dangling edge was found.
func checkDependencies(p *plan.Plan, r *Result) bool {
	defined := map[string]bool{}
	for _, m := range p.Milestones {
		defined[m.Slug] = true
	}
	dangling := false
	for _, m := range p.Milestones {
		for _, dep := range m.DependsOn {
			if !defined[dep] {
				dangling = true
				r.Errors = append(r.Errors, fmt.Sprintf(
					"dangling edge: milestone %q depends on %q, which no milestone defines", m.Slug, dep))
			}
		}
	}
	return dangling
}

// warnOverlappingPaths emits a non-fatal warning for each pair of independent
// milestones (no dependency path between them) whose contract paths globs
// overlap — a potential write-conflict a future concurrent executor would hit
// (plan-validation "Concurrently-runnable milestones with overlapping
// write-paths are warned").
func warnOverlappingPaths(p *plan.Plan, g *dag.Graph, r *Result) {
	for i := 0; i < len(p.Milestones); i++ {
		for j := i + 1; j < len(p.Milestones); j++ {
			a, b := p.Milestones[i], p.Milestones[j]
			if !g.Independent(a.Slug, b.Slug) {
				continue
			}
			for _, overlap := range overlappingGlobs(a.Contract.Paths, b.Contract.Paths) {
				r.Warnings = append(r.Warnings, fmt.Sprintf(
					"independent milestones %q and %q both write %q — potential concurrent write conflict",
					a.Slug, b.Slug, overlap))
			}
		}
	}
}

// overlappingGlobs returns the (deduplicated, sorted) set of path expressions
// whose matched file-sets could intersect across the two lists.
func overlappingGlobs(as, bs []string) []string {
	set := map[string]struct{}{}
	for _, a := range as {
		for _, b := range bs {
			if globsOverlap(a, b) {
				// Report the more specific (longer-base) glob so the message
				// points at the tightest overlapping expression.
				set[narrower(a, b)] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// globsOverlap reports whether two path globs could match a common file. It
// compares each glob's fixed base — the path components before the first
// component containing a wildcard — component-wise: the file-sets can intersect
// only when one base is a prefix of (or equal to) the other.
func globsOverlap(a, b string) bool {
	ba, bb := globBase(a), globBase(b)
	return isPrefix(ba, bb) || isPrefix(bb, ba)
}

// globBase splits a glob into the path components preceding its first wildcard
// component. A component holds a wildcard if it contains '*', '?', or '['.
func globBase(glob string) []string {
	var base []string
	for _, part := range strings.Split(strings.Trim(glob, "/"), "/") {
		if part == "" {
			continue
		}
		if strings.ContainsAny(part, "*?[") {
			break
		}
		base = append(base, part)
	}
	return base
}

// isPrefix reports whether short is a component-wise prefix of long.
func isPrefix(short, long []string) bool {
	if len(short) > len(long) {
		return false
	}
	for i := range short {
		if short[i] != long[i] {
			return false
		}
	}
	return true
}

// narrower returns whichever glob has the longer fixed base (the more specific
// of two overlapping expressions); ties return a.
func narrower(a, b string) string {
	if len(globBase(b)) > len(globBase(a)) {
		return b
	}
	return a
}

// Package resolve builds the machine-readable YAML projection of a plan
// (plan-projection "resolve reads a validated plan and emits a
// machine-readable YAML projection"): the full per-milestone data plus the
// authoritative depends_on edge set and a deterministic topological order.
//
// The projection is built from structs (never maps) so YAML key order is
// fixed, and every field that must survive round-tripping — including the
// polymorphic deliverables/criteria detail slots — is represented explicitly
// rather than mirrored as a scalar-or-mapping union, so the output is
// unambiguous and byte-reproducible across runs.
package resolve

import (
	"github.com/kentra-io/milestoned-plan-dag/internal/dag"
	"github.com/kentra-io/milestoned-plan-dag/internal/plan"

	"gopkg.in/yaml.v3"
)

// Projection is the full machine-readable view of a plan: its milestones in
// document order, each carrying its resolved depends_on edges, plus the
// computed topological order for the current serial executor.
type Projection struct {
	SchemaVersion string `yaml:"schemaVersion"`
	// Order is a valid topological order over the milestones (Kahn's
	// algorithm, ties broken by number — see internal/dag.TopoOrder),
	// deterministic and therefore byte-identical across runs.
	Order      []string              `yaml:"order"`
	Milestones []MilestoneProjection `yaml:"milestones"`
}

// MilestoneProjection is one milestone's full authored data plus its
// resolved (authoritative) incoming dependency edges.
type MilestoneProjection struct {
	Number       int          `yaml:"number"`
	Slug         string       `yaml:"slug"`
	Goal         string       `yaml:"goal"`
	Deliverables Deliverables `yaml:"deliverables"`
	Contract     Contract     `yaml:"contract"`
	Steps        []Step       `yaml:"steps"`
	// DependsOn is the resolved incoming edge set (explicit depends-on, or
	// the implicit preceding-milestone edge) — authoritative, and always
	// present even when empty (emitted as `[]`, never omitted).
	DependsOn []string `yaml:"depends_on"`
}

// Deliverables represents the prose-or-structured deliverables slot
// explicitly: exactly one of Prose or the Create/Modify/Test triple is
// populated, matching whichever shape the plan authored.
type Deliverables struct {
	Prose  string   `yaml:"prose,omitempty"`
	Create []string `yaml:"create,omitempty"`
	Modify []string `yaml:"modify,omitempty"`
	Test   []string `yaml:"test,omitempty"`
}

// Contract is a milestone's verification contract, carried verbatim.
type Contract struct {
	Check    string   `yaml:"check"`
	Criteria Criteria `yaml:"criteria"`
	Paths    []string `yaml:"paths"`
}

// Criteria represents the text-or-cases criteria slot explicitly.
type Criteria struct {
	Text  string     `yaml:"text,omitempty"`
	Cases []TestCase `yaml:"cases,omitempty"`
}

// TestCase is one entry of a test-case-shaped criteria.
type TestCase struct {
	Name  string `yaml:"name,omitempty"`
	Given string `yaml:"given,omitempty"`
	When  string `yaml:"when,omitempty"`
	Then  string `yaml:"then,omitempty"`
}

// Step is one checkbox-tracked breakdown item, with its checkbox state
// exposed as a field (the source ` [ ]`/`[x]` marker is not round-tripped).
type Step struct {
	Text  string   `yaml:"text"`
	Done  bool     `yaml:"done"`
	Files []string `yaml:"files,omitempty"`
}

// Resolve builds the projection for an already-validated plan: it resolves
// the dependency graph (internal/dag.Build), computes a deterministic
// topological order (internal/dag.TopoOrder — Kahn's algorithm, ties broken
// by number), and carries every milestone's full authored data alongside it.
//
// Resolve does not itself validate the plan; callers are expected to run
// internal/validate first (plan-projection: resolve operates on a validated
// plan). It returns an error only when the plan's depends-on relation
// contains a cycle, in which case no topological order exists.
func Resolve(p *plan.Plan) (*Projection, error) {
	g := dag.Build(p)
	order, err := g.TopoOrder()
	if err != nil {
		return nil, err
	}

	proj := &Projection{
		SchemaVersion: p.SchemaVersion,
		Order:         order,
		Milestones:    make([]MilestoneProjection, 0, len(p.Milestones)),
	}
	for _, m := range p.Milestones {
		proj.Milestones = append(proj.Milestones, milestoneProjection(m, g))
	}
	return proj, nil
}

// Marshal renders the projection as YAML — the sole projection format (no
// --format flag; plan-projection "YAML is the sole projection format").
func (p *Projection) Marshal() ([]byte, error) {
	return yaml.Marshal(p)
}

// milestoneProjection converts one authored milestone plus its resolved
// graph edges into its projection shape.
func milestoneProjection(m plan.Milestone, g *dag.Graph) MilestoneProjection {
	// DependsOn must always be non-nil so it marshals as `[]`, not `null`,
	// when a milestone has no incoming edges.
	dependsOn := append([]string{}, g.Deps[m.Slug]...)

	return MilestoneProjection{
		Number:       m.Number,
		Slug:         m.Slug,
		Goal:         m.Goal,
		Deliverables: deliverablesProjection(m.Deliverables),
		Contract:     contractProjection(m.Contract),
		Steps:        stepsProjection(m.Steps),
		DependsOn:    dependsOn,
	}
}

func deliverablesProjection(d plan.Deliverables) Deliverables {
	return Deliverables{
		Prose:  d.Prose,
		Create: d.Create,
		Modify: d.Modify,
		Test:   d.Test,
	}
}

func contractProjection(c plan.Contract) Contract {
	paths := append([]string{}, c.Paths...)
	return Contract{
		Check:    c.Check,
		Criteria: criteriaProjection(c.Criteria),
		Paths:    paths,
	}
}

func criteriaProjection(c plan.Criteria) Criteria {
	var cases []TestCase
	for _, tc := range c.Cases {
		cases = append(cases, TestCase{
			Name:  tc.Name,
			Given: tc.Given,
			When:  tc.When,
			Then:  tc.Then,
		})
	}
	return Criteria{
		Text:  c.Text,
		Cases: cases,
	}
}

func stepsProjection(steps []plan.Step) []Step {
	out := make([]Step, 0, len(steps))
	for _, s := range steps {
		out = append(out, Step{
			Text:  s.Text,
			Done:  s.Done,
			Files: s.Files,
		})
	}
	return out
}

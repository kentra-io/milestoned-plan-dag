package plan

import (
	"path/filepath"
	"testing"
)

const validDir = "../../testdata/valid"

func loadFixture(t *testing.T, name string) *Plan {
	t.Helper()
	p, err := LoadFile(filepath.Join(validDir, name))
	if err != nil {
		t.Fatalf("LoadFile(%s): %v", name, err)
	}
	return p
}

// plan-schema: "A plan stamps its schema version".
func TestSchemaVersionRoundTrips(t *testing.T) {
	p := loadFixture(t, "sequential.yaml")
	if p.SchemaVersion != "0.1.0" {
		t.Fatalf("schemaVersion = %q, want 0.1.0", p.SchemaVersion)
	}
}

// plan-schema: "Steps carry checkbox state".
func TestStepsCarryCheckboxState(t *testing.T) {
	p := loadFixture(t, "sequential.yaml")
	m2 := p.Milestones[1] // slug: model — steps [x] then [ ]
	if len(m2.Steps) != 2 {
		t.Fatalf("milestone 2 steps = %d, want 2", len(m2.Steps))
	}
	if !m2.Steps[0].Done {
		t.Errorf("step 0 done = false, want true ([x])")
	}
	if m2.Steps[1].Done {
		t.Errorf("step 1 done = true, want false ([ ])")
	}
	if m2.Steps[0].Text != "define the types" {
		t.Errorf("step 0 text = %q, want %q", m2.Steps[0].Text, "define the types")
	}
}

// plan-schema: "A plan omitting the optional slots is still valid".
func TestOptionalSlotsAbsentLoads(t *testing.T) {
	p := loadFixture(t, "optional-slots-absent.yaml")
	m := p.Milestones[0]
	if m.Deliverables.Structured() {
		t.Errorf("deliverables should be prose, not structured: %+v", m.Deliverables)
	}
	if m.Deliverables.Prose == "" {
		t.Errorf("prose deliverables should be populated")
	}
	if len(m.Contract.Criteria.Cases) != 0 {
		t.Errorf("criteria should be plain text, got %d cases", len(m.Contract.Criteria.Cases))
	}
	if m.Contract.Criteria.Text == "" {
		t.Errorf("plain criteria text should be populated")
	}
	for i, s := range m.Steps {
		if len(s.Files) != 0 {
			t.Errorf("step %d should have no files, got %v", i, s.Files)
		}
	}
}

func TestOptionalSlotsPresent(t *testing.T) {
	p := loadFixture(t, "optional-slots-present.yaml")
	m := p.Milestones[0]
	if !m.Deliverables.Structured() {
		t.Fatalf("deliverables should be structured, got %+v", m.Deliverables)
	}
	if len(m.Deliverables.Create) != 2 || len(m.Deliverables.Modify) != 1 || len(m.Deliverables.Test) != 1 {
		t.Errorf("structured deliverables = %+v", m.Deliverables)
	}
	if len(m.Contract.Criteria.Cases) != 2 {
		t.Fatalf("test-shaped criteria = %d cases, want 2", len(m.Contract.Criteria.Cases))
	}
	if m.Contract.Criteria.Cases[0].Then == "" {
		t.Errorf("test case then should be populated")
	}
	if len(m.Steps[0].Files) != 1 || m.Steps[0].Files[0] != "internal/plan/types.go" {
		t.Errorf("per-step files = %v", m.Steps[0].Files)
	}
	if !m.Steps[0].Done {
		t.Errorf("mapping step [x] should be done")
	}
}

// design D5: slug defaults to a derived kebab of the goal when omitted.
func TestSlugDerivedWhenOmitted(t *testing.T) {
	p := loadFixture(t, "slug-derived.yaml")
	if got, want := p.Milestones[0].Slug, "define-the-plan-model"; got != want {
		t.Fatalf("derived slug = %q, want %q", got, want)
	}
}

// design D5: an explicit slug is preserved verbatim.
func TestExplicitSlugPreserved(t *testing.T) {
	p := loadFixture(t, "branching.yaml")
	if got := p.Milestones[0].Slug; got != "a" {
		t.Fatalf("explicit slug = %q, want a", got)
	}
}

// plan-schema: "Explicit edges yield a branching DAG" — depends-on parsing.
func TestDependsOnParsed(t *testing.T) {
	p := loadFixture(t, "branching.yaml")
	if got := p.Milestones[1].DependsOn; len(got) != 1 || got[0] != "a" {
		t.Fatalf("milestone b depends-on = %v, want [a]", got)
	}
	if len(p.Milestones[0].DependsOn) != 0 {
		t.Errorf("milestone a should have no depends-on")
	}
}

// plan-schema: check accepts the `none` sentinel; paths accepts [].
func TestContractNoneAndEmptyPaths(t *testing.T) {
	none := loadFixture(t, "contract-none.yaml")
	if none.Milestones[0].Contract.Check != "none" {
		t.Errorf("check = %q, want none", none.Milestones[0].Contract.Check)
	}
	empty := loadFixture(t, "paths-empty.yaml")
	if paths := empty.Milestones[0].Contract.Paths; len(paths) != 0 {
		t.Errorf("paths = %v, want empty", paths)
	}
}

// The `# yaml-language-server:` editor header is tolerated (it is a comment).
func TestYAMLLanguageServerHeaderTolerated(t *testing.T) {
	// sequential.yaml carries the header on its first line.
	p := loadFixture(t, "sequential.yaml")
	if len(p.Milestones) != 3 {
		t.Fatalf("expected the header line to be ignored and 3 milestones parsed, got %d", len(p.Milestones))
	}
}

func TestDeriveSlug(t *testing.T) {
	cases := map[string]string{
		"Define the Plan Model!":            "define-the-plan-model",
		"  Leading/trailing  ":              "leading-trailing",
		"Already-kebab-slug":                "already-kebab-slug",
		"Numbers 123 and symbols @#$ mixed": "numbers-123-and-symbols-mixed",
	}
	for in, want := range cases {
		if got := DeriveSlug(in); got != want {
			t.Errorf("DeriveSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNonCheckboxStepRejected(t *testing.T) {
	_, err := Load([]byte(`schemaVersion: 0.1.0
milestones:
  - number: 1
    slug: bad
    goal: bad steps
    contract:
      check: none
      criteria: x
      paths: []
    steps:
      - just a plain bullet
`))
	if err == nil {
		t.Fatal("expected a plain (non-checkbox) step to be rejected at load")
	}
}

package resolve

import (
	"os"
	"testing"

	"milestoned-plan-dag/internal/plan"
)

// loadFixture is a small helper shared by the tests below.
func loadFixture(t *testing.T, path string) *plan.Plan {
	t.Helper()
	p, err := plan.LoadFile(path)
	if err != nil {
		t.Fatalf("load fixture %s: %v", path, err)
	}
	return p
}

// TestResolve_BranchingOrderAndEdges covers spec scenario "Independent
// milestones are linearized deterministically": a(1) has no deps, b(2) and
// c(3) both depend only on a. The order must place a first, then b before c
// (number tie-break), and depends_on must be present (authoritative edges).
func TestResolve_BranchingOrderAndEdges(t *testing.T) {
	p := loadFixture(t, "../../testdata/valid/branching.yaml")

	proj, err := Resolve(p)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	wantOrder := []string{"a", "b", "c"}
	if len(proj.Order) != len(wantOrder) {
		t.Fatalf("Order = %v, want %v", proj.Order, wantOrder)
	}
	for i, s := range wantOrder {
		if proj.Order[i] != s {
			t.Fatalf("Order = %v, want %v", proj.Order, wantOrder)
		}
	}

	byslug := map[string]MilestoneProjection{}
	for _, m := range proj.Milestones {
		byslug[m.Slug] = m
	}

	if got := byslug["a"].DependsOn; len(got) != 0 {
		t.Fatalf("a.DependsOn = %v, want empty (not omitted)", got)
	}
	if got := byslug["b"].DependsOn; len(got) != 1 || got[0] != "a" {
		t.Fatalf("b.DependsOn = %v, want [a]", got)
	}
	if got := byslug["c"].DependsOn; len(got) != 1 || got[0] != "a" {
		t.Fatalf("c.DependsOn = %v, want [a]", got)
	}
}

// TestResolve_CarriesFullMilestoneData covers spec scenario "A consumer
// reads contracts without parsing markdown": every milestone in the
// projection must expose slug, contract, and steps-with-checkbox-state.
func TestResolve_CarriesFullMilestoneData(t *testing.T) {
	p := loadFixture(t, "../../testdata/valid/branching.yaml")

	proj, err := Resolve(p)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	m := proj.Milestones[0]
	if m.Slug != "a" {
		t.Fatalf("Slug = %q, want %q", m.Slug, "a")
	}
	if m.Number != 1 {
		t.Fatalf("Number = %d, want 1", m.Number)
	}
	if m.Goal != "Foundation" {
		t.Fatalf("Goal = %q, want %q", m.Goal, "Foundation")
	}
	if m.Contract.Check != "none" {
		t.Fatalf("Contract.Check = %q, want %q", m.Contract.Check, "none")
	}
	if m.Contract.Criteria.Text != "Foundation is in place." {
		t.Fatalf("Contract.Criteria.Text = %q", m.Contract.Criteria.Text)
	}
	if len(m.Contract.Paths) != 1 || m.Contract.Paths[0] != "internal/foundation/**" {
		t.Fatalf("Contract.Paths = %v", m.Contract.Paths)
	}
	if len(m.Steps) != 1 || m.Steps[0].Text != "lay the foundation" || !m.Steps[0].Done {
		t.Fatalf("Steps = %+v, want one done step", m.Steps)
	}
}

// TestResolve_StructuredSlotsRoundTrip covers the structured deliverables and
// test-case-shaped criteria optional slots: both shapes must be represented
// faithfully in the projection.
func TestResolve_StructuredSlotsRoundTrip(t *testing.T) {
	p := loadFixture(t, "../../testdata/valid/optional-slots-present.yaml")

	proj, err := Resolve(p)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	m := proj.Milestones[0]
	if len(m.Deliverables.Create) != 2 || m.Deliverables.Create[0] != "internal/plan/types.go" {
		t.Fatalf("Deliverables.Create = %v", m.Deliverables.Create)
	}
	if len(m.Deliverables.Modify) != 1 || m.Deliverables.Modify[0] != "go.mod" {
		t.Fatalf("Deliverables.Modify = %v", m.Deliverables.Modify)
	}
	if len(m.Contract.Criteria.Cases) != 2 {
		t.Fatalf("Contract.Criteria.Cases = %v, want 2 cases", m.Contract.Criteria.Cases)
	}
	if m.Contract.Criteria.Cases[0].Name != "schema version round-trips" {
		t.Fatalf("Cases[0].Name = %q", m.Contract.Criteria.Cases[0].Name)
	}
	if len(m.Steps) != 2 || len(m.Steps[0].Files) != 1 || m.Steps[0].Files[0] != "internal/plan/types.go" {
		t.Fatalf("Steps = %+v", m.Steps)
	}
}

// TestProjection_MarshalDeterministic covers spec scenario "Independent
// milestones are linearized deterministically" (byte-reproducible across
// runs): marshaling the same projection twice must produce identical bytes.
func TestProjection_MarshalDeterministic(t *testing.T) {
	p := loadFixture(t, "../../testdata/valid/branching.yaml")

	proj, err := Resolve(p)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	first, err := proj.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	second, err := proj.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("Marshal is not deterministic:\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	// Re-resolving from scratch must also be byte-identical (determinism
	// holds across separate invocations, not just repeated marshaling of the
	// same struct).
	proj2, err := Resolve(p)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	third, err := proj2.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(first) != string(third) {
		t.Fatalf("Marshal is not deterministic across separate Resolve calls")
	}
}

// TestResolve_Golden pins the exact YAML projection shape against a
// committed fixture (plan-projection "A consumer reads contracts without
// parsing markdown").
func TestResolve_Golden(t *testing.T) {
	p := loadFixture(t, "../../testdata/valid/branching.yaml")

	proj, err := Resolve(p)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	got, err := proj.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	want, err := os.ReadFile("../../testdata/golden/branching.resolve.yaml")
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("resolve output does not match golden:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

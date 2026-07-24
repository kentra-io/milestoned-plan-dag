package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func read(t *testing.T, rel string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("../../testdata", rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return b
}

func validate(t *testing.T, rel string) *Result {
	t.Helper()
	r, err := Validate(read(t, rel))
	if err != nil {
		t.Fatalf("Validate(%s) returned parse error: %v", rel, err)
	}
	return r
}

func joinErrs(r *Result) string  { return strings.Join(r.Errors, "\n") }
func joinWarns(r *Result) string { return strings.Join(r.Warnings, "\n") }

// --- Accept cases -----------------------------------------------------------

func TestSequentialAccepted(t *testing.T) {
	r := validate(t, "valid/sequential.yaml")
	if !r.OK() {
		t.Fatalf("sequential should be accepted, errors: %s", joinErrs(r))
	}
	if len(r.Warnings) != 0 {
		t.Errorf("sequential should have no warnings, got: %s", joinWarns(r))
	}
}

func TestBranchingAccepted(t *testing.T) {
	r := validate(t, "valid/branching.yaml")
	if !r.OK() {
		t.Fatalf("branching should be accepted, errors: %s", joinErrs(r))
	}
	// b and c are independent but write disjoint paths — no warning.
	if len(r.Warnings) != 0 {
		t.Errorf("branching should have no overlap warning, got: %s", joinWarns(r))
	}
}

// plan-schema "An unverifiable milestone declares check none".
func TestContractNoneAccepted(t *testing.T) {
	if r := validate(t, "valid/contract-none.yaml"); !r.OK() {
		t.Fatalf("check: none should be well-formed, errors: %s", joinErrs(r))
	}
}

// plan-schema "An empty path set means an empty diff" — paths: [] is accepted.
func TestPathsEmptyAccepted(t *testing.T) {
	if r := validate(t, "valid/paths-empty.yaml"); !r.OK() {
		t.Fatalf("paths: [] should be well-formed, errors: %s", joinErrs(r))
	}
}

// --- Reject cases -----------------------------------------------------------

// plan-validation "A dependency cycle is rejected".
func TestCycleRejected(t *testing.T) {
	r := validate(t, "invalid/cycle.yaml")
	if r.OK() {
		t.Fatal("cycle plan should be rejected")
	}
	e := joinErrs(r)
	if !strings.Contains(e, "cycle") || !strings.Contains(e, "a") || !strings.Contains(e, "b") {
		t.Errorf("cycle error should name the cycle members a and b, got: %s", e)
	}
}

// plan-validation "A dangling edge is rejected".
func TestDanglingRejected(t *testing.T) {
	r := validate(t, "invalid/dangling.yaml")
	if r.OK() {
		t.Fatal("dangling plan should be rejected")
	}
	if !strings.Contains(joinErrs(r), "does-not-exist") {
		t.Errorf("dangling error should name the unresolved slug, got: %s", joinErrs(r))
	}
}

// plan-validation "A duplicate slug is rejected".
func TestDuplicateSlugRejected(t *testing.T) {
	r := validate(t, "invalid/duplicate-slug.yaml")
	if r.OK() {
		t.Fatal("duplicate-slug plan should be rejected")
	}
	e := joinErrs(r)
	if !strings.Contains(e, "duplicate slug") || !strings.Contains(e, "parse-dag") {
		t.Errorf("error should name the duplicated slug parse-dag, got: %s", e)
	}
}

func TestDuplicateNumberRejected(t *testing.T) {
	r := validate(t, "invalid/duplicate-number.yaml")
	if r.OK() {
		t.Fatal("duplicate-number plan should be rejected")
	}
	if !strings.Contains(joinErrs(r), "duplicate number") {
		t.Errorf("error should name the duplicated number, got: %s", joinErrs(r))
	}
}

// plan-validation "A milestone with no contract is rejected".
func TestMissingContractRejected(t *testing.T) {
	r := validate(t, "invalid/missing-contract.yaml")
	if r.OK() {
		t.Fatal("missing-contract plan should be rejected")
	}
	if !strings.Contains(joinErrs(r), "no-contract") {
		t.Errorf("error should name the milestone, got: %s", joinErrs(r))
	}
}

// plan-validation "An empty criteria is rejected".
func TestEmptyCriteriaRejected(t *testing.T) {
	r := validate(t, "invalid/empty-criteria.yaml")
	if r.OK() {
		t.Fatal("empty-criteria plan should be rejected")
	}
	e := joinErrs(r)
	if !strings.Contains(e, "blank-criteria") || !strings.Contains(e, "criteria") {
		t.Errorf("error should name the milestone and cite criteria, got: %s", e)
	}
}

// --- Warn case --------------------------------------------------------------

// plan-validation "Overlapping paths on independent milestones warn but pass".
func TestOverlappingPathsWarnButPass(t *testing.T) {
	r := validate(t, "warn/overlapping-paths.yaml")
	if !r.OK() {
		t.Fatalf("overlapping-paths plan should still pass, errors: %s", joinErrs(r))
	}
	if len(r.Warnings) == 0 {
		t.Fatal("overlapping-paths plan should emit a warning")
	}
	w := joinWarns(r)
	if !strings.Contains(w, "b") || !strings.Contains(w, "c") || !strings.Contains(w, "internal/foo/**") {
		t.Errorf("warning should name b, c, and the overlapping glob, got: %s", w)
	}
}

// --- Glob-overlap unit coverage --------------------------------------------

func TestGlobsOverlap(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"internal/foo/**", "internal/foo/**", true},
		{"internal/**", "internal/foo/**", true},
		{"internal/foo/**", "internal/foo/bar.go", true},
		{"internal/b/**", "internal/c/**", false},
		{"go.mod", "cmd/**", false},
		{"go.mod", "go.mod", true},
		{"**", "anything/at/all", true},
	}
	for _, c := range cases {
		if got := globsOverlap(c.a, c.b); got != c.want {
			t.Errorf("globsOverlap(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

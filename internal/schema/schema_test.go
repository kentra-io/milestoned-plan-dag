package schema

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	validDir      = "../../testdata/valid"
	publishedPath = "../../schema/plan.schema.json"
)

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}

func validFixtures(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(validDir)
	if err != nil {
		t.Fatalf("read testdata/valid: %v", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".yaml" {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		t.Fatal("no valid fixtures found")
	}
	return names
}

// The published draft-2020-12 schema shape-validates every valid fixture.
func TestValidFixturesShapeValidate(t *testing.T) {
	for _, name := range validFixtures(t) {
		name := name
		t.Run(name, func(t *testing.T) {
			if err := Validate(readFile(t, filepath.Join(validDir, name))); err != nil {
				t.Fatalf("Validate(%s): %v", name, err)
			}
		})
	}
}

// The embedded copy must byte-match the published schema (drift guard).
func TestEmbeddedMatchesPublished(t *testing.T) {
	published := readFile(t, publishedPath)
	embedded := PlanSchemaJSON()
	if string(published) != string(embedded) {
		t.Fatalf("embedded internal/schema/plan.schema.json differs from published schema/plan.schema.json")
	}
}

// The embedded schema compiles as a valid draft-2020-12 schema.
func TestSchemaCompiles(t *testing.T) {
	if _, err := Compiled(); err != nil {
		t.Fatalf("Compiled(): %v", err)
	}
}

// A plan lacking schemaVersion fails shape validation.
func TestMissingSchemaVersionRejected(t *testing.T) {
	doc := []byte(`milestones:
  - number: 1
    slug: a
    goal: no schema version
    contract:
      check: none
      criteria: x
      paths: []
    steps:
      - "[ ] do a thing"
`)
	if err := Validate(doc); err == nil {
		t.Fatal("expected a plan lacking schemaVersion to fail shape validation")
	}
}

// A malformed schemaVersion (v-prefixed) fails the semver pattern.
func TestVPrefixedSchemaVersionRejected(t *testing.T) {
	doc := []byte(`schemaVersion: v0.1.0
milestones:
  - number: 1
    slug: a
    goal: v prefixed
    contract:
      check: none
      criteria: x
      paths: []
    steps:
      - "[ ] do a thing"
`)
	if err := Validate(doc); err == nil {
		t.Fatal("expected a v-prefixed schemaVersion to fail shape validation")
	}
}

// A contract missing paths fails shape validation.
func TestContractMissingPathsRejected(t *testing.T) {
	doc := []byte(`schemaVersion: 0.1.0
milestones:
  - number: 1
    slug: a
    goal: no paths
    contract:
      check: none
      criteria: x
    steps:
      - "[ ] do a thing"
`)
	if err := Validate(doc); err == nil {
		t.Fatal("expected a contract with no paths key to fail shape validation")
	}
}

// A plain-bullet (non-checkbox) step fails the step pattern.
func TestNonCheckboxStepRejected(t *testing.T) {
	doc := []byte(`schemaVersion: 0.1.0
milestones:
  - number: 1
    slug: a
    goal: bad step
    contract:
      check: none
      criteria: x
      paths: []
    steps:
      - just a plain bullet
`)
	if err := Validate(doc); err == nil {
		t.Fatal("expected a non-checkbox step to fail shape validation")
	}
}

// The optional-slots-absent fixture still validates.
func TestOptionalSlotsAbsentValidates(t *testing.T) {
	if err := Validate(readFile(t, filepath.Join(validDir, "optional-slots-absent.yaml"))); err != nil {
		t.Fatalf("optional-slots-absent should validate: %v", err)
	}
}

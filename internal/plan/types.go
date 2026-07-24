// Package plan defines the canonical, machine-first plan model — a DAG of
// verifiable milestones — and loads it from YAML.
//
// YAML is the single source of truth (design D1). This package owns the typed
// in-memory model and the loader; JSON-Schema shape validation lives in
// internal/schema, and the semantic DAG rules land in later milestones.
package plan

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Plan is a complete plan: a schema-versioned list of milestones.
type Plan struct {
	// SchemaVersion is semantic-versioned without a "v" prefix (e.g. 0.1.0),
	// so consumers can detect the format revision (plan-schema: "The plan
	// declares a schema version").
	SchemaVersion string `yaml:"schemaVersion"`
	// Milestones are in document order; index 0 is the first milestone.
	Milestones []Milestone `yaml:"milestones"`
}

// Milestone is one verifiable unit of the plan.
type Milestone struct {
	// Number is the milestone's ordinal reading order and deterministic
	// tie-break.
	Number int `yaml:"number"`
	// Slug is the stable, position-independent, kebab-case identity that
	// dependency edges reference. It is explicit but defaults to a derived
	// kebab-slug of Goal when omitted (design D5); the loader fills it in.
	Slug string `yaml:"slug,omitempty"`
	// Goal is the milestone's human label / intent.
	Goal string `yaml:"goal"`
	// Deliverables is either prose or a structured create/modify/test map
	// (optional structured detail slot).
	Deliverables Deliverables `yaml:"deliverables,omitempty"`
	// Contract is the mandatory verification contract.
	Contract Contract `yaml:"contract"`
	// Steps are checkbox-tracked breakdown items.
	Steps []Step `yaml:"steps"`
	// DependsOn lists the slugs of milestones that must complete before this
	// one. Empty means an implicit dependency on the preceding milestone
	// (resolved in a later milestone).
	DependsOn []string `yaml:"depends-on,omitempty"`
}

// Contract is a milestone's mandatory verification contract.
type Contract struct {
	// Check is a single executable command, or the sentinel "none" for
	// genuinely unverifiable work. It MUST be present.
	Check string `yaml:"check"`
	// Criteria is non-empty pass/fail text, or a test-case-shaped list
	// (optional structured detail slot).
	Criteria Criteria `yaml:"criteria"`
	// Paths is the allowed write-set. An empty list means the diff must be
	// empty; a "**" wildcard means the diff is consciously unconfined.
	Paths []string `yaml:"paths"`
}

// Deliverables is either prose (Prose set) or a structured create/modify/test
// file list. A plan may omit deliverables entirely (IsZero reports that).
type Deliverables struct {
	Prose  string   `yaml:"-"`
	Create []string `yaml:"create,omitempty"`
	Modify []string `yaml:"modify,omitempty"`
	Test   []string `yaml:"test,omitempty"`
}

// IsZero reports whether no deliverables were authored.
func (d Deliverables) IsZero() bool {
	return d.Prose == "" && len(d.Create) == 0 && len(d.Modify) == 0 && len(d.Test) == 0
}

// Structured reports whether the deliverables use the create/modify/test slot.
func (d Deliverables) Structured() bool {
	return len(d.Create) > 0 || len(d.Modify) > 0 || len(d.Test) > 0
}

// UnmarshalYAML accepts either a scalar prose string or a
// create/modify/test mapping.
func (d *Deliverables) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		return value.Decode(&d.Prose)
	case yaml.MappingNode:
		var aux struct {
			Create []string `yaml:"create"`
			Modify []string `yaml:"modify"`
			Test   []string `yaml:"test"`
		}
		if err := value.Decode(&aux); err != nil {
			return err
		}
		d.Create, d.Modify, d.Test = aux.Create, aux.Modify, aux.Test
		return nil
	default:
		return fmt.Errorf("deliverables: expected prose text or a create/modify/test mapping, got %s", nodeKind(value.Kind))
	}
}

// Criteria is either plain-language pass/fail text (Text set) or a
// test-case-shaped list (Cases set, an optional structured detail slot).
type Criteria struct {
	Text  string     `yaml:"-"`
	Cases []TestCase `yaml:"-"`
}

// TestCase is one entry of a test-case-shaped criteria.
type TestCase struct {
	Name  string `yaml:"name,omitempty"`
	Given string `yaml:"given,omitempty"`
	When  string `yaml:"when,omitempty"`
	Then  string `yaml:"then,omitempty"`
}

// UnmarshalYAML accepts either scalar prose or a sequence of test cases.
func (c *Criteria) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		return value.Decode(&c.Text)
	case yaml.SequenceNode:
		return value.Decode(&c.Cases)
	default:
		return fmt.Errorf("criteria: expected plain-language text or a list of test cases, got %s", nodeKind(value.Kind))
	}
}

// Step is one checkbox-tracked breakdown item. Done reflects [x]/[X]; Text is
// the item text with the checkbox marker stripped. Files is the optional
// per-step file reference (an optional structured detail slot).
type Step struct {
	Text  string   `yaml:"-"`
	Done  bool     `yaml:"-"`
	Files []string `yaml:"files,omitempty"`
}

// UnmarshalYAML accepts either a "[ ] text" / "[x] text" checkbox string or a
// {text, files} mapping whose text carries the checkbox marker.
func (s *Step) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		var raw string
		if err := value.Decode(&raw); err != nil {
			return err
		}
		return s.parseCheckbox(raw)
	case yaml.MappingNode:
		var aux struct {
			Text  string   `yaml:"text"`
			Files []string `yaml:"files"`
		}
		if err := value.Decode(&aux); err != nil {
			return err
		}
		s.Files = aux.Files
		return s.parseCheckbox(aux.Text)
	default:
		return fmt.Errorf("step: expected a checkbox string or a {text, files} mapping, got %s", nodeKind(value.Kind))
	}
}

// parseCheckbox splits a "[ ]"/"[x]" prefix from the step text.
func (s *Step) parseCheckbox(raw string) error {
	t := strings.TrimSpace(raw)
	switch {
	case strings.HasPrefix(t, "[x]"), strings.HasPrefix(t, "[X]"):
		s.Done = true
		s.Text = strings.TrimSpace(t[3:])
	case strings.HasPrefix(t, "[ ]"):
		s.Done = false
		s.Text = strings.TrimSpace(t[3:])
	default:
		return fmt.Errorf("step %q must be a checkbox item beginning with [ ] or [x]", raw)
	}
	return nil
}

// nodeKind renders a yaml.Kind for error messages.
func nodeKind(k yaml.Kind) string {
	switch k {
	case yaml.ScalarNode:
		return "a scalar"
	case yaml.MappingNode:
		return "a mapping"
	case yaml.SequenceNode:
		return "a sequence"
	case yaml.AliasNode:
		return "an alias"
	case yaml.DocumentNode:
		return "a document"
	default:
		return "an unknown node"
	}
}

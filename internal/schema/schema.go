// Package schema embeds the published draft-2020-12 JSON Schema and shape-
// validates a plan YAML document against it.
//
// This is the shape tier of the two-tier validation (design D3): the JSON
// Schema covers structure for external consumers and live editor validation;
// the semantic DAG rules a JSON Schema cannot express land in a later milestone.
//
// Go's go:embed cannot reference a file outside the embedding package's
// directory tree, so the published schema at ../../schema/plan.schema.json is
// mirrored here as plan.schema.json and embedded. TestEmbeddedMatchesPublished
// guards the two copies against drift.
package schema

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

//go:embed plan.schema.json
var planSchemaJSON []byte

// fallbackResourceURI identifies the embedded schema when it carries no $id
// (it always does — TestSchemaIDIsThePinnedPublishedURL guards that — so this
// is defensive only). It must be absolute: the compiler resolves a relative
// resource name against the process working directory, which would stamp a
// synthetic file:///<cwd>/plan.schema.json into every validation error — a
// path that does not exist and that nothing ever tried to read. The schema is
// embedded; nothing is fetched at runtime, so the URI is pure identity and
// must not depend on where the CLI was invoked from.
const fallbackResourceURI = "https://raw.githubusercontent.com/kentra-io/milestoned-plan-dag/main/schema/plan.schema.json"

// PlanSchemaJSON returns the embedded draft-2020-12 JSON Schema bytes.
func PlanSchemaJSON() []byte {
	out := make([]byte, len(planSchemaJSON))
	copy(out, planSchemaJSON)
	return out
}

// Compiled returns the compiled embedded plan schema.
func Compiled() (*jsonschema.Schema, error) {
	return compileFrom(planSchemaJSON)
}

// Validate shape-validates a raw plan YAML document against the embedded
// draft-2020-12 JSON Schema, returning a non-nil error naming the shape
// violation when the document does not conform.
func Validate(planYAML []byte) error {
	sch, err := Compiled()
	if err != nil {
		return err
	}
	inst, err := toInstance(planYAML)
	if err != nil {
		return err
	}
	if err := sch.Validate(inst); err != nil {
		return shapeViolation(err)
	}
	return nil
}

// compileFrom compiles a JSON Schema from its raw bytes as draft 2020-12.
func compileFrom(schemaJSON []byte) (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("parse json schema: %w", err)
	}
	uri := resourceURI(doc)
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	if err := c.AddResource(uri, doc); err != nil {
		return nil, fmt.Errorf("add schema resource: %w", err)
	}
	sch, err := c.Compile(uri)
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	return sch, nil
}

// resourceURI is the absolute URI the schema is registered and compiled under:
// its own $id, so the schema names itself exactly once (in the published JSON),
// with fallbackResourceURI covering an $id-less document.
func resourceURI(doc any) string {
	if obj, ok := doc.(map[string]any); ok {
		if id, ok := obj["$id"].(string); ok && id != "" {
			return id
		}
	}
	return fallbackResourceURI
}

// shapeViolation restates a jsonschema failure as a statement about the plan.
// The library's root line — "jsonschema validation failed with '<uri>#'" —
// names the schema, which reads like a failed schema load and tells a plan
// author nothing actionable: the binary validates against the one schema it
// embeds. The causes beneath it are the actionable part, so they are kept
// verbatim, re-indented to preserve their nesting under the new root.
func shapeViolation(err error) error {
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) || len(ve.Causes) == 0 {
		return err
	}
	lines := make([]string, 0, len(ve.Causes))
	for _, cause := range ve.Causes {
		lines = append(lines, "- "+strings.ReplaceAll(cause.Error(), "\n", "\n  "))
	}
	return fmt.Errorf("plan does not conform to the plan schema:\n%s", strings.Join(lines, "\n"))
}

// toInstance decodes plan YAML into a JSON-typed instance suitable for
// jsonschema validation (numbers as json.Number, mappings as map[string]any).
func toInstance(planYAML []byte) (any, error) {
	var doc any
	if err := yaml.Unmarshal(planYAML, &doc); err != nil {
		return nil, fmt.Errorf("parse plan yaml: %w", err)
	}
	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("normalize plan to json: %w", err)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("decode plan instance: %w", err)
	}
	return inst, nil
}

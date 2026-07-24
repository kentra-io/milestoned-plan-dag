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
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
)

//go:embed plan.schema.json
var planSchemaJSON []byte

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
	return sch.Validate(inst)
}

// compileFrom compiles a JSON Schema from its raw bytes as draft 2020-12.
func compileFrom(schemaJSON []byte) (*jsonschema.Schema, error) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return nil, fmt.Errorf("parse json schema: %w", err)
	}
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2020)
	if err := c.AddResource("plan.schema.json", doc); err != nil {
		return nil, fmt.Errorf("add schema resource: %w", err)
	}
	sch, err := c.Compile("plan.schema.json")
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	return sch, nil
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

# milestoned-plan-dag — Implementation Plan

The primitive is built in six milestones. This scaffold milestone stands up the
repo; later milestones add the model, validator, projections, skills, and
integration. Each milestone carries a checkable validation contract.

## Milestone 1 — Repo scaffold (this milestone)
Stand up a framework-neutral, MIT-licensed, single-binary Go repo with the
primitive shape: `go.mod` (module `milestoned-plan-dag`), `LICENSE` (MIT),
`README.md`, a `cmd/milestoned-plan-dag/main.go` argument-dispatch skeleton for
`validate` / `render` / `resolve` stubs, an empty `internal/` package skeleton,
this in-repo `spec.md` + `implementation-plan.md`, and a `.github/workflows/ci.yml`
running build + vet + test.
- Contract: `go build ./... && go vet ./...` compiles clean; MIT `LICENSE`,
  `spec.md`, and `implementation-plan.md` present; no non-Go language-runtime
  dependency. Foundational — makes no spec scenario pass directly.

## Milestone 2 — Plan data model, YAML loader, and published JSON Schema
Define the canonical YAML plan format as typed Go plus a published draft-2020-12
JSON Schema, and load + shape-validate a plan file. Adds `internal/plan/` (types
and loader), `schema/plan.schema.json`, `internal/schema/` (embeds + shape
validation), and `testdata/valid/` fixtures.
- Realizes plan-schema: schemaVersion stamping, checkbox step state, optional
  slots omittable.

## Milestone 3 — `validate`: DAG resolution, semantic checks, path-overlap warning
Implement `validate`: shape validation plus semantic DAG guarantees (implicit
document-order chain, cycle / dangling / duplicate / malformed-contract
rejection) and the non-fatal overlapping-write-paths warning. Adds
`internal/dag/`, `internal/validate/`, the wired `validate` command, and
`testdata/invalid/` + `testdata/warn/` fixtures.
- Realizes plan-validation and the DAG-resolution parts of plan-schema.

## Milestone 4 — Projection: `resolve` (machine YAML) and `render` (human markdown)
Emit the machine YAML projection with authoritative edges + a deterministic
topological order (`resolve`), and a read-only markdown view (`render`). Adds
`internal/resolve/`, `internal/render/`, both commands wired, and golden fixtures.
- Realizes plan-projection, including determinism and the retired `--format json`.

## Milestone 5 — Authoring skills
Ship agent-agnostic authoring skill(s) that guide writing a valid YAML plan
against the schema, with a worked example that passes `validate`. Adds
`skills/plan-author/`.

## Milestone 6 — Integrate as a submodule + record consumer follow-ons
Register the finished primitive as a git submodule of the wrapper repo, list it in
the primitive registry, and record the two out-of-scope consumer reworks as linked
follow-ons.

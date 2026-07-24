# milestoned-plan-dag — Specification

A framework-neutral primitive owning a **machine-first YAML plan format**: a plan
is a **DAG of verifiable milestones**. YAML is the single source of truth; a
published JSON Schema validates its shape; a Go CLI is authoritative for the
semantic DAG rules a JSON Schema cannot express. Markdown is a read-only
projection generated from the YAML — nothing parses markdown back.

This document distills the three capability specs the primitive realizes —
`plan-schema`, `plan-validation`, and `plan-projection`.

## Capability: plan-schema (the authored plan format)

### Milestone identity
Every milestone SHALL carry a stable, kebab-case `slug` and an ordinal `number`.
The `slug` is the identity that dependency edges reference and is
position-independent, so reordering milestones does not change any edge; the
`number` gives a reading order and a deterministic tie-break. A `slug` SHALL be
unique within a plan. The `slug` is explicit but defaults to a derived kebab-slug
of the milestone title when omitted.

### Dependency edges form a DAG
A milestone MAY declare `depends-on` listing the slugs of milestones that MUST
complete before it. The relation is a directed acyclic graph of "must complete
before" edges; `depends-on` is the only edge type. A milestone that declares no
`depends-on` inherits an implicit dependency on the immediately preceding
milestone in document order — so a plan with no explicit edges is a sequential
chain, and an existing sequential plan stays valid unchanged.

### Every milestone carries a verification contract
Every milestone SHALL carry a verification `contract` with three fields:
- `check` — a single executable command, or the sentinel `none` for genuinely
  unverifiable work. It MUST be present so absence is never silent.
- `criteria` — non-empty plain-language pass/fail text a judge can evaluate.
- `paths` — the milestone's allowed write-set. An empty list (`[]`) means the
  diff MUST be empty; a `**` wildcard means the diff is consciously unconfined.

### Steps are checkbox-tracked
A milestone's steps SHALL be authored as checkbox items (`[ ]` / `[x]`) so step
completion is structurally visible and a downstream consumer can refuse to treat
a plan as complete while tracked steps remain unchecked.

### The plan declares a schema version
A plan SHALL declare a `schemaVersion` using semantic versioning without a `v`
prefix, beginning at `0.1.0`, so the format can evolve without silently breaking
consumers.

### Optional structured detail slots
The schema reserves optional slots that let a plan carry file-specific detail
without mandating it: structured `Create` / `Modify` / `Test` deliverables, a
per-step `files` reference, and a test-case-shaped `criteria`. A plan omitting all
of these remains valid.

## Capability: plan-validation (the validator guarantees)

`validate` runs JSON-Schema shape validation, then semantic validation. It
rejects (non-zero exit, naming the offender):
- a `depends-on` cycle (a valid plan always admits at least one topological order);
- a `depends-on` entry referencing a slug no milestone defines (dangling edge);
- two milestones sharing a `slug` or a `number` (duplicate identity);
- a missing or malformed contract — a `check` that is neither a command nor the
  `none` sentinel, an empty `criteria`, or an absent `paths` key.

It accepts `check: none` and `paths: []` as well-formed. It **warns** — not fails —
when two milestones with no dependency path between them (potentially runnable
concurrently by a future executor) declare overlapping `paths`, naming both
milestones and the overlapping glob, while still exiting 0.

## Capability: plan-projection (machine + human views)

`resolve` reads a validated plan and emits a machine-readable **YAML** projection
exposing, per milestone: `number`, `slug`, `goal`, `deliverables`, the full
`contract` (`check` / `criteria` / `paths`), `steps` with checkbox state, and
`depends_on` edges. The `depends_on` edges are authoritative; the projection
additionally carries a computed valid **topological order** for the current serial
executor, deterministic with ties broken by `number` so the output is
byte-reproducible across runs. YAML is the sole projection format — there is no
`--format json` flag.

`render` emits a read-only human markdown view (Goal / Deliverables / Validation
contract / checkbox steps) from the same model; nothing parses it back.

## Non-goals

- **Concurrent execution.** The schema and projection are DAG-capable so a future
  consumer *can* run independent milestones together, but no concurrent-execution
  engine is built here; the executor stays serial.
- **Non-verifiable or non-decomposable work.** Every milestone is verifiable and
  diff-confined. The `check: none` / `paths: []` escapes cover legitimate edge
  cases within the methodology; they are not a route to opting out of
  verification wholesale.

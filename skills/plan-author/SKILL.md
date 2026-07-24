---
name: plan-author
description: Author and validate a milestoned-plan-dag YAML plan — a DAG of verifiable milestones with a mandatory check/criteria/paths contract. Use when writing a new plan.yaml, editing milestones/dependencies in an existing plan, or debugging a validate/resolve/render failure for this format.
---

# Authoring a milestoned-plan-dag plan

A plan is a **DAG of verifiable milestones** written as a single YAML document. It
is machine-first: every milestone carries an identity, a goal, and a mandatory
verification contract, so the plan can be validated, topologically ordered, and
rendered to human-readable markdown without any further interpretation.

This skill documents the grammar and the CLI workflow for authoring one. It does
not assume or name any particular agent runtime — the plan format and CLI are
agent-agnostic.

## Scope: one plan = one git repository

A plan describes work in **a single repository**. Every `contract.paths` glob is
relative to one project root, so the plan has exactly one root and every
milestone's write-set resolves against it.

**Multi-repo and multi-module (separate git roots) projects are not supported.**
Do not write a plan whose milestones deliver into different repositories, and do
not path-prefix milestones to reach a sibling repo or a nested checkout — there
is no way to express which root a path belongs to, and executors (which commit
per milestone at the single root) will silently commit nothing for the
milestones that point elsewhere. Write one plan per repository instead.

The same rule covers a plan whose deliverable is a **new** repository: create the
repo first, then write the plan inside it.

## Top-level shape

```yaml
schemaVersion: 0.1.0
milestones:
  - number: 1
    ...
```

- `schemaVersion` — semver, no `v` prefix. The current format version is `0.1.0`.
- `milestones` — a non-empty list, in document order (index 0 is read first, but
  identity and ordering are governed by `number`/`slug` and `depends-on`, not
  list position).

## Milestone fields

Each entry under `milestones` is a mapping:

| Field | Required | Meaning |
|---|---|---|
| `number` | yes | Integer ordinal (≥ 1), unique across the plan. Reading order / deterministic tie-break. |
| `slug` | no | Stable, position-independent kebab-case identity (`^[a-z0-9]+(-[a-z0-9]+)*$`). If omitted, it is derived from `goal` (lowercased, punctuation stripped, words joined with `-`). `depends-on` always references slugs, never numbers. |
| `goal` | yes | The milestone's human label / intent (non-empty string). |
| `deliverables` | no | Either a prose string, or a structured `create` / `modify` / `test` map of file lists (an optional structured detail slot — pick whichever fits). |
| `contract` | yes | The mandatory verification contract — see below. |
| `steps` | yes | A non-empty list of checkbox items — see below. |
| `depends-on` | no | A list of slugs this milestone depends on. |

### Identity: `number` + `slug`

`number` and `slug` must each be unique across the plan — duplicates of either
are rejected. `slug` is the identity that `depends-on` edges reference, so it is
what stays stable if milestones are reordered or renumbered later. You may omit
`slug` and let the loader derive one from `goal`; give an explicit `slug` when
you want a shorter or more stable name than the derived one.

### The DAG and the implicit-chain fallback

Milestones form a DAG via `depends-on` (a list of slugs):

```yaml
depends-on:
  - some-earlier-slug
```

**A milestone with no `depends-on` gets an implicit edge on the immediately
preceding milestone** (in document order). This means:

- A plan where no milestone ever declares `depends-on` is simply a **sequential
  chain** — the degenerate, single-branch DAG. This is the common case; most
  plans don't need explicit edges at all.
- Declaring `depends-on` on one or more milestones turns the plan into a
  **branching DAG** — e.g. two milestones can both depend on the same earlier
  "foundation" milestone and then proceed independently of each other.

Validation rejects: dangling edges (a `depends-on` slug nothing defines) and
cycles (named by the offending milestone chain).

### The contract: `check` / `criteria` / `paths`

Every milestone **must** carry a `contract` — this is the format's core
guarantee: nothing ships without a stated way to check it. All three keys are
mandatory (though their values may use the "conscious escape" forms below);
omitting the `contract` block, or leaving any of the three keys unset, is
rejected.

```yaml
contract:
  check: go test ./internal/foo/...
  criteria: The new package's tests pass and cover the branch case.
  paths:
    - internal/foo/**
```

- **`check`** — a single executable command that proves the milestone's work,
  OR the sentinel **`none`** for a milestone whose outcome genuinely has no
  automated proof (e.g. a design note or a decision record) — a *conscious*
  escape, not a default. `check` must not be blank.
- **`criteria`** — the pass/fail statement a verifier judges the check (or the
  unverifiable work, when `check: none`) against. It must be non-empty. It may
  instead be a structured list of test cases, each with `then` required and
  `name` / `given` / `when` optional:

  ```yaml
  criteria:
    - name: the loader accepts a minimal plan
      given: a plan with one milestone and no depends-on
      when: it is loaded
      then: no error is returned
  ```
- **`paths`** — the allowed write-set for this milestone, as a list of globs.
  It is required but may be the **empty list `paths: []`** — a conscious escape
  meaning this milestone's diff must be empty (a verify-only or read-only
  milestone). A single **`**`** entry is a conscious escape meaning the
  write-set is deliberately unconfined. Ordinary entries are directory globs,
  e.g. `internal/foo/**`, always relative to the plan's single project root
  (see "Scope" above — they cannot reach another repository).

  Two *independent* milestones (no dependency path between them, so a future
  concurrent executor could run them together) whose `paths` globs could match
  the same file produce a non-fatal **warning** (not a rejection) — a signal to
  either add a `depends-on` edge or narrow the globs.

### Steps

`steps` is a non-empty list of checkbox items, each either a bare checkbox
string or a `{text, files}` mapping when you want to record which files a step
touches:

```yaml
steps:
  - "[x] scaffold the package"
  - "[ ] wire it into the CLI"
  - text: "[ ] write the loader"
    files:
      - internal/plan/load.go
```

Every entry must begin with `[ ]` (not started) or `[x]` / `[X]` (done); the
loader strips the marker and records a boolean `done` flag plus the remaining
text.

## Authoring workflow

1. **Write the YAML** by hand or by editing an existing plan, following the
   grammar above. Pin the editor's schema helper to the published JSON Schema
   with the first line of the file:

   ```yaml
   # yaml-language-server: $schema=../path/to/schema/plan.schema.json
   ```

2. **Validate.** This is the authoritative check — it runs JSON-Schema shape
   validation, then the semantic DAG rules a schema can't express (unique
   identity, resolved edges, acyclicity, well-formed contracts), and warns on
   overlapping write-paths between independent milestones:

   ```
   milestoned-plan-dag validate plan.yaml
   ```

   Exit code `0` on a clean or warning-only plan, `1` on rejection (each error
   is printed, naming the offending milestone).

3. **Resolve**, once the plan validates, to see the machine-readable
   projection — every milestone plus the authoritative `depends_on` edges
   (implicit edges made explicit) and the computed topological `order`:

   ```
   milestoned-plan-dag resolve plan.yaml
   ```

4. **Render**, to get a read-only human markdown view (one section per
   milestone, with a fenced `contract` block and checkbox steps) for review:

   ```
   milestoned-plan-dag render plan.yaml
   ```

Iterate steps 1–2 until `validate` exits `0`; `resolve`/`render` are safe to
run at any point after that to sanity-check the plan reads the way you intend.

## See also

- `skills/plan-author/example.yaml` — a worked branching plan exercising this
  grammar end to end.
- `schema/plan.schema.json` — the published JSON Schema (shape validation and
  editor `$schema` wiring).

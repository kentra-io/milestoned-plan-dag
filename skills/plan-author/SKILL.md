---
name: plan-author
description: Writes and validates milestoned-plan-dag plans — YAML DAGs of milestones an implementer can execute without exploring the codebase. Use when writing, editing or reviewing a plan.yaml.
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

## The bar: an implementer never explores

A milestone is handed to an implementer **alone** — this repository, the
plan-level `description` and `rules`, this one milestone, nothing else. It
must start work without opening another document, searching the codebase to
learn conventions, or asking a question. Write every milestone so a skilled
engineer with zero context for this codebase and questionable taste could
finish it.

- **Names, not references.** `deliverables` lists every file path, and the
  milestone's `description` or steps name each type or function the file
  introduces with its signature. "As described in the design doc" is a
  plan failure.
- **Code, not intentions.** A step that changes code shows the code. A step
  that runs a command shows the command and its expected output.
- **Criteria written out.** Each criterion states `given`/`when`/`then` in full
  and `name`s the test that proves it.
- **Cross-milestone types are named once, where they are born.** A type,
  function or file one milestone creates and another uses is named with its
  signature in the milestone that creates it. Later milestones use exactly
  that name.
- **Versions are pinned.** A dependency is added with an exact version in the
  earliest milestone that needs it. "Latest" is not a deliverable.
- **Every milestone ends green** on the standard `check`, and keeps every
  earlier milestone's tests green.

Test it: read one milestone with the rest of the plan covered. If you would
have to look something up or ask before starting, the milestone is not done.

## Rules: references the implementer loads

`rules` exist so the implementer loads the right guidance up front and the
verifier grades against a known list.

- List **references, never copies**: a skill name (`java-hexagonal`) or a repo
  path (`constitution/constitution.md`, `docs/naming.md`).
- Put what every milestone needs in plan-level `rules`; put only additions in a
  milestone's `rules`.
- Leave out what the agent runtime loads by itself — for Claude Code,
  `CLAUDE.md`/`AGENTS.md` and path-scoped `.claude/rules/*.md`.
- Keep the list minimal and relevant. Every reference costs the implementer
  context; a rule that does not bear on the milestone's files does not belong.

## Parallel milestones

Two milestones with no dependency path between them may run at the same time,
in separate worktrees, and are merged afterwards. Give them non-overlapping
`paths` (`validate` warns when they could touch the same file) and put shared
types, shared files and dependency pins in a milestone both depend on.

## Top-level shape

```yaml
schemaVersion: 0.3.0
description: |          # optional — context every milestone shares
  ...
rules:                  # optional — references every milestone loads
  - <skill name or repo path>
milestones:
  - number: 1
    ...
```

- `schemaVersion` — semver, no `v` prefix. The current format version is `0.3.0`; `validate` rejects any other.
- `description` — optional markdown: context every milestone shares.
- `rules` — optional list of references (skill names or repo paths) every
  milestone loads.
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
| `description` | no | Markdown: why this milestone exists and how to build it — the decisions, the shape of the code, the gotchas. Context for the implementer, not a restatement of the steps. |
| `rules` | no | References (skill names or repo paths) this milestone needs beyond the plan-level `rules`. |
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

### The DAG

Milestones form a DAG via `depends-on` (a list of slugs):

```yaml
depends-on:
  - some-earlier-slug
```

Every edge is written out. **A milestone without `depends-on` is a root** — it
can start at once:

- Milestones that can start in parallel, such as two independent foundations,
  each declare no `depends-on`.
- A milestone that needs another's output lists that milestone's slug. A
  sequential plan lists the predecessor on every milestone after the first.
- A plan may end in several sinks; nothing requires a joining milestone.

Validation rejects: dangling edges (a `depends-on` slug nothing defines) and
cycles (named by the offending milestone chain), and a `schemaVersion` other
than `0.3.0`.

### The contract: `check` / `criteria` / `paths`

Every milestone **must** carry a `contract` — this is the format's core
guarantee: nothing ships without a stated way to check it. All three keys are
mandatory (though their values may use the "conscious escape" forms below);
omitting the `contract` block, or leaving any of the three keys unset, is
rejected.

```yaml
contract:
  check: go test ./...
  criteria:
    - name: internal/foo/branch_test.go::TestHandlesEmptyInput
      given: an empty input
      when: the new branch case runs
      then: it returns the zero value with no error
  paths:
    - internal/foo/**
```

- **`check`** — the command that proves the milestone did not break the
  repository: normally **the project's standard validation command**, the one a
  maintainer runs before pushing (`./gradlew clean test`, `go test ./...`,
  `npm test`). The same command on every milestone is the expected shape, not a
  smell — it proves nothing regressed *and* that this milestone's new tests ran
  inside the real suite rather than under an isolating filter. Per-milestone
  specificity belongs in `criteria`, which names the tests. The sentinel
  **`none`** is for a milestone whose outcome genuinely has no automated proof
  (a design note, a decision record) — a *conscious* escape, not a default.
  `check` must not be blank.
- **`criteria`** — the pass/fail statement a verifier grades the milestone
  against. Prefer the structured list: one entry per behaviour this milestone
  delivers, `then` required, `name` carrying the **test identity** (the file and
  test name that proves it), `given`/`when` written out in full:

  ```yaml
  criteria:
    - name: internal/plan/load_test.go::TestLoadRejectsDuplicateSlug
      given: a plan with two milestones sharing a slug
      when: it is loaded
      then: loading fails and names the duplicated slug
  ```

  A bare string is allowed and stays valid, but it gives the verifier nothing to
  check the milestone *did* anything — reach for it only where there is genuinely
  no test to name. `criteria` must be non-empty.
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

`steps` is a non-empty list of checkbox items. Each step is one action of two
to five minutes: write the failing test, run it and see it fail, write the
code, run it and see it pass, commit. A step may span several lines — use a
YAML block scalar and start it with the checkbox:

```yaml
steps:
  - |
    [ ] Write the failing test
    ```go
    func TestHandlesEmptyInput(t *testing.T) {
        if got := Parse(""); got != nil {
            t.Fatalf("Parse(\"\") = %v, want nil", got)
        }
    }
    ```
  - "[ ] Run `go test ./internal/foo/ -run TestHandlesEmptyInput` — expect FAIL: undefined: Parse"
  - text: "[ ] Implement Parse"
    files:
      - internal/foo/parse.go
```

Every entry must begin with `[ ]` (not started) or `[x]` / `[X]` (done); the
loader strips the marker and records a boolean `done` flag plus the remaining
text.

## Authoring workflow

1. **Write the YAML** by hand or by editing an existing plan, following the
   grammar above. Point the editor's schema helper at the published JSON Schema
   with the first line of the file, pinned to the release tag matching the CLI
   you validate with (`milestoned-plan-dag --version`):

   ```yaml
   # yaml-language-server: $schema=https://raw.githubusercontent.com/kentra-io/milestoned-plan-dag/v0.3.0/schema/plan.schema.json
   ```

   The tag matters. The CLI validates against the schema **embedded in the
   binary** — it never fetches this URL, and the directive is an editor hint
   only. Pinned to `main` (or to a tag other than your binary's), the editor can
   go green on a plan the CLI rejects. `validate` is always the authority.

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
   (exactly the declared edges, `[]` for a root) and the computed topological `order`:

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

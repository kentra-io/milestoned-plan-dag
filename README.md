# milestoned-plan-dag

A framework-neutral, single-binary primitive for a **machine-first YAML plan
format**: a plan is a **DAG of verifiable milestones**. Each milestone carries a
stable slug, an ordinal number, a goal, deliverables, checkbox-tracked steps, and
a mandatory verification **contract** (`check` / `criteria` / `paths`). Milestones
declare dependencies with `depends-on`; a milestone with no `depends-on` inherits
an implicit edge on the preceding milestone, so a plan with no explicit edges is a
sequential chain (the degenerate DAG).

The format is authored in YAML and validated against a published JSON Schema; the
CLI is authoritative for the semantic DAG rules a JSON Schema cannot express.

## Install

```sh
brew install kentra-io/tap/milestoned-plan-dag
```

Or `go install github.com/kentra-io/milestoned-plan-dag/cmd/milestoned-plan-dag@latest`,
or grab a per-platform archive from the
[releases page](https://github.com/kentra-io/milestoned-plan-dag/releases).
Full install and release documentation: [`docs/releasing.md`](./docs/releasing.md).

> **Using this with [`spec-lifecycle`](https://github.com/kentra-io/spec-lifecycle)?**
> `lifecycle` shells out to this binary for its plan-stage gate and archive
> step-completion gate, resolving it by the exact name `milestoned-plan-dag` on
> `PATH`. Install both, or neither.

## CLI

A single static binary with three subcommands:

- `validate <plan.yaml>` — shape validation plus semantic DAG guarantees
  (cycle / dangling-edge / duplicate / malformed-contract rejection; a non-fatal
  overlapping-write-paths warning for independent milestones).
- `resolve <plan.yaml>` — emit the machine-readable YAML projection: per-milestone
  data plus authoritative `depends_on` edges and a deterministic topological order.
- `render <plan.yaml>` — emit a read-only human markdown view of the plan.

```
milestoned-plan-dag <validate|resolve|render> <plan.yaml>
```

All three subcommands are implemented: `validate` runs shape + semantic DAG
checks, `resolve` emits the machine-readable YAML projection (authoritative
`depends_on` edges + a deterministic topological order), and `render` emits a
read-only human markdown view.

## Authoring a plan

[`skills/plan-author/SKILL.md`](./skills/plan-author/SKILL.md) is the
agent-agnostic authoring skill: it documents the full YAML grammar (milestone
identity, the `depends-on` DAG and its implicit-chain fallback, the mandatory
`check`/`criteria`/`paths` contract and its conscious escapes, checkbox steps)
and the `validate` → `resolve`/`render` CLI workflow. See
[`skills/plan-author/example.yaml`](./skills/plan-author/example.yaml) for a
worked, validating branching plan.

Point an editor's YAML language server at the published JSON Schema for
inline completion and validation, pinned to the release tag matching your
installed CLI (`milestoned-plan-dag --version`):

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/kentra-io/milestoned-plan-dag/v0.1.1/schema/plan.schema.json
```

This is an editor hint only — the CLI validates against the schema embedded in
the binary and never fetches anything. Pin the two to the same tag, or the
editor can accept a plan `validate` rejects. (Plans living inside this repo can
use a relative path to `schema/plan.schema.json` instead.)

## Repository shape

- `cmd/milestoned-plan-dag/` — CLI entrypoint and argument dispatch.
- `internal/` — implementation packages (populated in later milestones).
- `schema/` — the published JSON Schema (added in a later milestone).
- `skills/` — agent-agnostic authoring skills (added in a later milestone).
- `spec.md` — the in-repo specification of the plan format, validator, and
  projection contract.
- `implementation-plan.md` — the milestone build plan.

## License

MIT. See [`LICENSE`](./LICENSE).

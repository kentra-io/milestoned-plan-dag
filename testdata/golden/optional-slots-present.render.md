## Milestone 1: Exercise every optional structured detail slot

**Goal** — Exercise every optional structured detail slot

**Deliverables** — create: internal/plan/types.go, internal/plan/load.go; modify: go.mod; test: internal/plan/load_test.go

**Validation contract**

```contract
check: go test ./internal/plan/...
criteria:
  - schema version round-trips: given a plan authored against the initial format, when it is loaded, then it declares schemaVersion 0.1.0
  - structured deliverables round-trip: given a milestone with create, modify and test deliverables, when it is loaded, then each list is populated
paths:
  - internal/plan/**
```

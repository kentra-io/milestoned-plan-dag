## Milestone 1: Exercise every optional structured detail slot

**Goal** — Exercise every optional structured detail slot

**Deliverables** — create: internal/plan/types.go, internal/plan/load.go; modify: go.mod; test: internal/plan/load_test.go

**Validation contract**

```contract
check: go test ./internal/plan/...
criteria:
  - schema version round-trips: given a plan authored against the initial format, when it is loaded, then it declares schemaVersion 0.1.0
  - steps carry checkbox state: given a milestone with checkbox steps, when it is loaded, then each step's done flag reflects its [ ] or [x] marker
paths:
  - internal/plan/**
```

**Steps**

- [x] define the model
- [ ] write the loader

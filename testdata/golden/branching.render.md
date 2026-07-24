## Milestone 1: Foundation

**Goal** — Foundation

**Deliverables** — The shared foundation both branches build on.

**Validation contract**

```contract
check: none
criteria: Foundation is in place.
paths:
  - internal/foundation/**
```

**Steps**

- [x] lay the foundation

## Milestone 2: First branch

**Goal** — First branch

**Deliverables** — The first independent branch.

**Validation contract**

```contract
check: go test ./internal/b/...
criteria: Branch b works on top of the foundation.
paths:
  - internal/b/**
```

**Steps**

- [ ] build branch b

## Milestone 3: Second branch

**Goal** — Second branch

**Deliverables** — The second independent branch.

**Validation contract**

```contract
check: go test ./internal/c/...
criteria: Branch c works on top of the foundation.
paths:
  - internal/c/**
```

**Steps**

- [ ] build branch c

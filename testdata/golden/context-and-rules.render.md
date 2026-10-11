## Plan

Adds a greeting endpoint. Every milestone follows the hexagonal layout.

**Rules** — java-hexagonal, constitution/constitution.md

## Milestone 1: Greeter domain service

**Goal** — Greeter domain service

**Description**

The domain service the adapter in milestone 2 calls.

**Rules** — docs/naming.md

**Deliverables** — create: src/main/java/greeter/Greeter.java; test: src/test/java/greeter/GreeterTest.java

**Validation contract**

```contract
check: ./gradlew test
criteria:
  - src/test/java/greeter/GreeterTest.java::greetsByName: given a Greeter, when greet("Ada") is called, then it returns "Hello, Ada"
paths:
  - src/main/java/greeter/**
  - src/test/java/greeter/**
```

**Steps**

- [ ] Write the failing test
  ```java
  assertThat(new Greeter().greet("Ada")).isEqualTo("Hello, Ada");
  ```
- [ ] Implement Greeter.greet

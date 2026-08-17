# Installing & releasing `milestoned-plan-dag`

## Install

### Homebrew (macOS / Linux)

```sh
brew install kentra-io/tap/milestoned-plan-dag
```

The cask is published to [`kentra-io/homebrew-tap`](https://github.com/kentra-io/homebrew-tap)
by the release pipeline — the same tap that carries the
[`constitution`](https://github.com/kentra-io/adr-sourced-constitution) and
[`lifecycle`](https://github.com/kentra-io/spec-lifecycle) casks. The binary is
not code-signed; the cask strips the macOS quarantine attribute on install so it
runs without a Gatekeeper prompt.

> **Installing this alongside `lifecycle`?** `spec-lifecycle` shells out to this
> binary for its plan-stage gate and archive step-completion gate, resolving it
> by the exact name `milestoned-plan-dag` on `PATH`. Install both, or neither —
> `lifecycle init` warns when this binary is absent, and the plan/archive gates
> fail outright without it.

### `go install`

```sh
go install github.com/kentra-io/milestoned-plan-dag/cmd/milestoned-plan-dag@latest
```

`go install` builds locally, so `milestoned-plan-dag --version` reports the
module pseudo-version + VCS revision rather than a release tag (both are wired
in `cmd/milestoned-plan-dag/version.go`).

### Release archive (direct download)

Every release publishes per-platform archives plus a `checksums.txt`. Asset
names follow a fixed template:

```
milestoned-plan-dag_<version>_<os>_<arch>.tar.gz     # linux, darwin
milestoned-plan-dag_<version>_<os>_<arch>.zip        # windows
```

where `<version>` is the tag without its leading `v` (tag `v0.1.0` -> `0.1.0`).
So the linux/amd64 tarball for `v0.1.0` is at the deterministic URL:

```
https://github.com/kentra-io/milestoned-plan-dag/releases/download/v0.1.0/milestoned-plan-dag_0.1.0_linux_amd64.tar.gz
```

Each archive also carries the published JSON Schema (`schema/*.json`) so an
editor's YAML language server can be pointed at it without cloning the repo.

### claudebox / Docker

`milestoned-plan-dag` is a single static Go binary (`CGO_ENABLED=0`) with no
runtime dependencies. Because the asset URL is deterministic, a container image
can install a pinned version with a plain download-and-extract:

```dockerfile
# Install the milestoned-plan-dag CLI at a pinned version. Static binary —
# nothing else to provision.
ARG PLAN_DAG_VERSION=0.1.0
RUN set -eux; \
    arch="$(dpkg --print-architecture)"; \
    case "$arch" in \
      amd64) goarch=amd64 ;; \
      arm64) goarch=arm64 ;; \
      *) echo "unsupported arch: $arch" >&2; exit 1 ;; \
    esac; \
    base="https://github.com/kentra-io/milestoned-plan-dag/releases/download/v${PLAN_DAG_VERSION}"; \
    asset="milestoned-plan-dag_${PLAN_DAG_VERSION}_linux_${goarch}.tar.gz"; \
    cd /tmp; \
    curl -fsSL "$base/$asset" -o "$asset"; \
    curl -fsSL "$base/checksums.txt" -o checksums.txt; \
    # Verify the download against the release checksums before extracting.
    # --ignore-missing checks only the asset we fetched, not every line.
    sha256sum -c --ignore-missing checksums.txt; \
    tar -xzf "$asset" -C /usr/local/bin milestoned-plan-dag; \
    rm "$asset" checksums.txt; \
    milestoned-plan-dag --version
```

The `<version>_<os>_<arch>` template is produced by the `archives.name_template`
in [`.goreleaser.yaml`](../.goreleaser.yaml); keep the two in sync if either
changes.

## The `--version` output is a cross-repo contract

`spec-lifecycle` runs `milestoned-plan-dag --version` and parses the result with
the regexp `^milestoned-plan-dag version\s+(.*)$` (its
`internal/plandag/version.go`) to drive its companion-primitive preflight. The
`milestoned-plan-dag version ` prefix is therefore load-bearing.
`cmd/milestoned-plan-dag/version_test.go` holds a verbatim copy of that regexp
and fails if the format drifts — fix the output rather than relaxing the test.

## How a release is cut

Releases are fully automated by [`.github/workflows/release.yml`](../.github/workflows/release.yml),
which triggers on any `v*` tag.

**Prerequisites (one-time, owner-side):**

- `kentra-io/milestoned-plan-dag` exists (public) and the bot has write access.
- `kentra-io/homebrew-tap` exists (public) — already established by the
  constitution and spec-lifecycle casks.
- `HOMEBREW_TAP_TOKEN` is an **org-level** Actions secret (the fine-grained PAT
  needs `Contents: read/write` on `homebrew-tap`). GoReleaser needs it to push
  the cask cross-repo — the default `GITHUB_TOKEN` is scoped to this repo only.

**Cutting a release:**

First bump the pinned schema URL to the version about to be released. The
schema's `$id`, its embedded mirror, and every `$schema` directive in the docs
and the authoring skill all carry the tag, so an editor validates against
exactly the schema the released binary embeds:

```sh
prev=v0.1.1 next=v0.2.0   # the pin as it stands, and the version being cut
grep -rl "milestoned-plan-dag/$prev/schema/plan.schema.json" \
  --include='*.json' --include='*.md' --include='*.yaml' . |
  xargs sed -i '' "s|milestoned-plan-dag/$prev/schema|milestoned-plan-dag/$next/schema|g"
go test ./internal/schema/   # TestPinnedSchemaURLsAgree: every mention agrees
```

Commit that, then tag:

```sh
git tag "$next"
git push origin "$next"
```

The release workflow re-checks the pin against the tag and refuses to publish a
mismatch, so a forgotten bump fails loudly instead of shipping a schema that
misidentifies itself.

CI then runs `goreleaser release --clean`, which:

1. Builds all targets (linux/darwin/windows × amd64/arm64).
2. Creates the GitHub Release with the archives and `checksums.txt`.
3. Pushes the updated Homebrew cask to `kentra-io/homebrew-tap` (the third cask
   in that tap, alongside `constitution` and `lifecycle`).

Validate the config locally before tagging:

```sh
goreleaser check                                      # config is valid
goreleaser release --snapshot --clean --skip=publish  # dry run, builds everything into ./dist
```

`./dist` is git-ignored; snapshot artifacts are never committed.

## If a release fails midway

A tag push that fails partway (e.g. the cask push errors after the GitHub
Release was created) leaves a partial release and a published tag. GoReleaser
does not overwrite an existing release, so re-running against the same tag will
not recover it — tear the partial state down, fix the cause, and re-tag:

```sh
gh release delete v0.1.0 --yes             # remove the partial GitHub Release
git push --delete origin v0.1.0            # remove the remote tag
git tag -d v0.1.0                          # remove the local tag
# ...fix the cause (config, secret, etc.), then re-cut:
git tag v0.1.0
git push origin v0.1.0
```

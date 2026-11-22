# Contributing

## Setup

You need Go 1.26+, [Task](https://taskfile.dev), [golangci-lint](https://golangci-lint.run) v2,
and Docker (for e2e tests and schema updates). GoReleaser is pinned in `tools/go.mod`
(separate module; it needs a newer Go, which `GOTOOLCHAIN=auto` downloads), oapi-codegen
as a `tool` in the root `go.mod`.

```sh
task            # list tasks
task build      # ./bin/sinkctl
task test       # unit tests
task lint
task e2e        # starts the pinned Bugsink container, runs ./e2e
```

## Layout

| Path | Purpose |
|---|---|
| `cmd/sinkctl` | `main` |
| `internal/ops` | Operations as typed input/output funcs, plus the error kinds. All logic lives here. |
| `internal/cli` | cobra commands: thin adapters over `ops` |
| `internal/mcp` | MCP tools: thin adapters over `ops` |
| `internal/output` | table / JSON / YAML rendering |
| `internal/config` | contexts and env resolution |
| `internal/bugsink` | generated API client, do not edit |
| `api/openapi.yaml` | vendored Bugsink schema |
| `e2e/` | tests against a real Bugsink (`-tags e2e`) |

New features go into `ops` first, then get wired into both the CLI and MCP.

## Updating the Bugsink API

The Bugsink version is pinned in `e2e/Dockerfile` (Dependabot bumps it).
After a bump:

```sh
task schema     # fetches api/openapi.yaml from that image and regenerates the client
```

Commit both `api/openapi.yaml` and `internal/bugsink/client.gen.go`. CI fails if
either is stale.

## Commits and releases

Use [Conventional Commits](https://www.conventionalcommits.org) (`feat:`, `fix:`, `docs:`, ...);
the release changelog groups on these prefixes.

Maintainers release by pushing a tag:

```sh
git tag v0.1.0 && git push origin v0.1.0
```

The `Release` workflow builds, signs, and publishes with GoReleaser.
Run `task snapshot` to try the build locally.

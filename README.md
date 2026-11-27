# sinkctl

Command-line client and MCP server for [Bugsink](https://www.bugsink.com), built on
Bugsink's canonical REST API (`/api/canonical/0/`).

> **Status:** early development. Read-only commands for projects, issues and events.

## Install

Download a binary for your platform from
[Releases](https://github.com/kidk/sinkctl/releases), or build from source:

```sh
go install github.com/kidk/sinkctl/cmd/sinkctl@latest
```

### Nix

```sh
nix run github:kidk/sinkctl -- version
```

In a flake, add the input and use the package or the overlay:

```nix
inputs.sinkctl = {
  url = "github:kidk/sinkctl"; # or pin a tag: github:kidk/sinkctl/vX.Y.Z (flake exists from v0.1.1)
  inputs.nixpkgs.follows = "nixpkgs";
};

# packages
environment.systemPackages = [ inputs.sinkctl.packages.${pkgs.stdenv.hostPlatform.system}.default ];
# or: nixpkgs.overlays = [ inputs.sinkctl.overlays.default ]; then pkgs.sinkctl
```

Nix builds report version `dev` plus the exact commit. `follows` needs a nixpkgs
with Go 1.26 or newer.

### Verify a release

Each release ships `checksums.txt`, signed keylessly with cosign, an SBOM per
archive, and a GitHub build provenance attestation.

```sh
cosign verify-blob \
  --certificate-identity "https://github.com/kidk/sinkctl/.github/workflows/release.yml@refs/tags/vX.Y.Z" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --bundle checksums.txt.sigstore.json checksums.txt
sha256sum --check --ignore-missing checksums.txt

gh attestation verify sinkctl_X.Y.Z_linux_amd64.tar.gz --repo kidk/sinkctl
```

## Configure

Create an API token in Bugsink, then save it as a named context:

```sh
sinkctl config login prod --url https://bugsink.example.com   # prompts for the token
echo "$TOKEN" | sinkctl config login staging --url https://bugsink.staging.example.com --token-stdin
sinkctl config use prod
```

`login` checks the token against Bugsink before saving (skip with `--no-verify`)
and switches to the new context. Or set environment variables:

```sh
export BUGSINK_URL=https://bugsink.example.com
export BUGSINK_TOKEN=...
```

Contexts live in `~/.config/sinkctl/config.yaml` (respects `$XDG_CONFIG_HOME`,
written with mode 0600), which you can also edit by hand. `config login` rewrites
the file, so comments are not kept:

```yaml
current-context: prod
contexts:
  prod:
    url: https://bugsink.example.com
    token: ...
  staging:
    url: https://bugsink.staging.example.com
    token: ...
```

Pick a context per command with `--context staging`. `BUGSINK_URL` and
`BUGSINK_TOKEN` override values from the selected context.

## Commands

| Command | MCP tool |
|---|---|
| `projects list [--team UUID]` | `projects_list` |
| `projects get <id>` | `projects_get` |
| `projects event-counts <id> [--start T] [--end T]` | `projects_event_counts` |
| `issues list [--project N] [-q TEXT] [--state S] [--sort S] [--order asc\|desc]` | `issues_list` |
| `issues get <uuid\|friendly-id>` | `issues_get` |
| `issues event-counts <id> [--start T] [--end T]` | `issues_event_counts` |
| `events list --issue UUID [-q TEXT] [--order asc\|desc]` | `events_list` |
| `events get <uuid>` | `events_get` |
| `events stacktrace <uuid>` | `events_stacktrace` |
| `events lookup --project N --event-id UUID` | `events_lookup` |

List commands return one page plus `next_cursor`; pass it back with `--cursor`
(MCP: `cursor`). The CLI also has `--all` to follow every page. Times are RFC 3339
UTC hours, like `2026-01-02T15:00:00Z`. `events stacktrace` prints Markdown in a
terminal.

## Output

`-o auto` (default) prints a table in a terminal and JSON when piped, so
scripts and agents get JSON without extra flags. Force a format with
`-o json`, `-o yaml` or `-o table`.

## Errors and exit codes

| Exit | Kind | Meaning |
|---|---|---|
| 0 | | success |
| 1 | `error` | unclassified failure |
| 2 | `usage` | bad command, flag or argument |
| 3 | `auth` | missing/invalid credentials or unknown context |
| 4 | `not_found` | resource does not exist |
| 5 | `api`, `network` | Bugsink returned an error or was unreachable |
| 130 | | interrupted (Ctrl-C) |

When output is JSON (piped, or `-o json`), errors go to stderr as
`{"error":{"kind":"not_found","message":"...","status":404}}`. MCP tools return
the same object as the text of an error result.

## MCP server

`sinkctl mcp` serves the same operations as MCP tools over stdio. Example
client config:

```json
{
  "mcpServers": {
    "bugsink": {
      "command": "sinkctl",
      "args": ["mcp"],
      "env": { "BUGSINK_URL": "https://bugsink.example.com", "BUGSINK_TOKEN": "..." }
    }
  }
}
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[Apache-2.0](LICENSE)

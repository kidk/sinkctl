# sinkctl

Command-line client and MCP server for [Bugsink](https://www.bugsink.com), built on
Bugsink's canonical REST API (`/api/canonical/0/`).

> **Status:** early development. Only `version` and `mcp` exist so far.

## Install

Download a binary for your platform from
[Releases](https://github.com/kidk/sinkctl/releases), or build from source:

```sh
go install github.com/kidk/sinkctl/cmd/sinkctl@latest
```

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

Create an API token in Bugsink, then either set environment variables:

```sh
export BUGSINK_URL=https://bugsink.example.com
export BUGSINK_TOKEN=...
```

or write `~/.config/sinkctl/config.yaml` (respects `$XDG_CONFIG_HOME`, keep it `chmod 600`):

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

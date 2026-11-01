# kanboard-cli

> **Unofficial, community-built, not affiliated with Kanboard**

`kanboard` is a small Go CLI for the [Kanboard](https://github.com/kanboard/kanboard) JSON-RPC API. It is **unofficial, community-built, not affiliated with Kanboard**. It targets humans and AI agents that want structured JSON (or tables) instead of wiring an MCP server — see [kanboard/kanboard#5827](https://github.com/kanboard/kanboard/issues/5827).

MIT licensed. Third-party notices in [NOTICE](NOTICE); release archives also ship `THIRD_PARTY_NOTICES` and `third_party_licenses/`.

## Install

Or install from source with Go **1.26.10** (matches `go.mod`):

```bash
go install github.com/swtormy-agents/kanboard-cli/cmd/kanboard@latest
```

## Quick start

```bash
export KANBOARD_URL=https://kanboard.example.com
export KANBOARD_API_TOKEN=...    # application API token (user jsonrpc) or personal API token
# export KANBOARD_USERNAME=...   # personal API only (required for kanboard me)

kanboard health
kanboard projects list
kanboard tasks list --project 1
kanboard search --project 1 --query 'status:open'
```

## Output

- Default: JSON on stdout; one JSON error object on stderr
- `--output table` for simple tabular views
- `--jq '...'` to filter JSON (gojq)
- `--pretty` to indent JSON

## Safety

- Never log or print API tokens; `--dry-run` and errors redact credentials as `<credentials>` / `<API_TOKEN>`
- Agents stay read-only. Mutating commands and non-read-only `rpc` methods need `--write`, a TTY, and typing `write` when prompted
- Exit codes: `0` ok, `1` runtime/API, `2` usage or config, `3` auth, `4` blocked

See [docs/api-scope.md](docs/api-scope.md) for JSON-RPC methods covered in v0.1.

## Agent skills

Bundled skill (also in [skills/kanboard/SKILL.md](skills/kanboard/SKILL.md)):

```bash
kanboard skill install --agent cursor
kanboard skill print
```

## License

MIT — see [LICENSE](LICENSE).

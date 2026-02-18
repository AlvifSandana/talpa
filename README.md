# Talpa

[![CI](https://github.com/AlvifSandana/talpa/actions/workflows/ci.yml/badge.svg)](https://github.com/AlvifSandana/talpa/actions/workflows/ci.yml)
[![Release](https://github.com/AlvifSandana/talpa/actions/workflows/release.yml/badge.svg)](https://github.com/AlvifSandana/talpa/actions/workflows/release.yml)
[![Go Version](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)](https://go.dev/)
[![Platform](https://img.shields.io/badge/platform-linux--first-lightgrey)](#compatibility)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Talpa** is a Linux-first, single-binary CLI/TUI for safe system cleanup, disk analysis, project artifact purging, and host status inspection.

It is designed for developers and power users who want one tool with consistent safety controls instead of stitching together many ad-hoc maintenance commands.

---

## Table of Contents

- [Why Talpa](#why-talpa)
- [Current Capabilities](#current-capabilities)
- [Quick Start](#quick-start)
- [Usage](#usage)
- [Commands](#commands)
- [Safety Model](#safety-model)
- [JSON Output for Automation](#json-output-for-automation)
- [Practical Examples](#practical-examples)
- [Troubleshooting](#troubleshooting)
- [Compatibility](#compatibility)
- [Development](#development)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [Security](#security)
- [License](#license)

## Why Talpa

Linux maintenance is often fragmented across multiple utilities. Talpa unifies common maintenance flows into a single command surface with safety-first defaults:

- Explicit dry-run behavior for destructive workflows
- Centralized path/symlink safety validation
- Layered confirmations for high-risk operations
- Local JSONL operation logs for auditability
- JSON output mode for scripting and CI integrations

## Current Capabilities

Implemented command set:

- `clean` — clean safe cache and temporary candidates
- `analyze` — inspect disk usage and optionally `trash`/`delete` targets
- `purge` — purge project build artifacts
- `status` — show host/system snapshot (including watch mode)
- `completion` — generate shell completion scripts
- `update` — plan self-update workflow
- `remove` — plan self-remove workflow
- `uninstall` — uninstall app + Talpa-related leftovers
- `installer` — clean installer artifacts
- `optimize` — execute safe optimization workflow

> Notes:
> - `talpa` without arguments opens interactive mode on supported TTY terminals.
> - On non-interactive terminals, `talpa` prints help.

## Quick Start

### Option A — Download release binary

Artifacts are published in GitHub Releases and include checksums:

- `talpa-linux-amd64`
- `talpa-linux-arm64`
- `SHA256SUMS`

Example verification flow:

```bash
sha256sum -c SHA256SUMS
chmod +x talpa-linux-amd64
./talpa-linux-amd64 --help
```

### Option B — Build from source

Prerequisite: **Go 1.22**.

```bash
git clone https://github.com/AlvifSandana/talpa.git
cd talpa
go build -o talpa ./cmd/talpa
./talpa --help
```

## Usage

Interactive mode:

```bash
talpa
```

CLI mode:

```bash
talpa clean --dry-run
talpa analyze ~ --action inspect
talpa purge --paths ~/Projects,~/Code --dry-run
talpa status --watch --interval 2
```

## Commands

| Command | Purpose | Key Flags |
| --- | --- | --- |
| `talpa clean` | Safe cleanup candidates | `--system` |
| `talpa analyze [path]` | Disk tree analysis + action mode | `--depth`, `--limit`, `--sort`, `--min-size`, `--query`, `--only-candidates`, `--action inspect\|trash\|delete` |
| `talpa purge` | Purge project artifacts | `--paths`, `--depth`, `--recent-days` |
| `talpa status` | Host snapshot / live watch | `--top`, `--interval`, `--watch` |
| `talpa completion <shell>` | Shell completions | `bash`, `zsh`, `fish`, `powershell` |
| `talpa update` | Plan self-update | _(global flags)_ |
| `talpa remove` | Plan self-remove | _(global flags)_ |
| `talpa uninstall` | Uninstall app/leftovers | `--apply`, `--target backend:name` |
| `talpa installer` | Cleanup installer artifacts | `--apply` |
| `talpa optimize` | Safe optimization workflow | `--apply` |

### Global Flags

- `--dry-run` — preview actions without mutating files
- `--debug` — verbose debugging output
- `--yes` — non-interactive confirmation
- `--confirm HIGH-RISK` — second confirmation token for high-risk apply flows
- `--json` — JSON output mode
- `--no-oplog` — disable operation logging

## Safety Model

Talpa enforces a centralized safety gate for destructive operations.

Core protections:

- Path validation (null/control/traversal/clean-path checks)
- Hard-block critical paths (e.g. `/`, `/usr`, `/etc`, `/proc`, `/sys`, `/dev`, `/run`)
- Symlink escape protection and scope enforcement
- High-risk layered confirmation policy (`--yes` + `--confirm HIGH-RISK`)
- Local JSONL operation logs (disabled via `--no-oplog` or `TALPA_NO_OPLOG=1`)
- No telemetry by default

Security and safety references:

- [`docs/SAFETY_MODEL.md`](docs/SAFETY_MODEL.md)
- [`docs/OPERATIONS_LOG.md`](docs/OPERATIONS_LOG.md)

### Important Limitation

`analyze --action trash` has stronger fd-based safety semantics on Unix platforms. On non-Unix platforms, fallback behavior is best-effort rename with reduced guarantees.

## JSON Output for Automation

Use `--json` to integrate Talpa with scripts, CI jobs, and dashboards.

```bash
talpa clean --dry-run --json
talpa analyze /home/user --action inspect --json
```

Schema documentation:

- [`docs/JSON_SCHEMA.md`](docs/JSON_SCHEMA.md)

## Practical Examples

### 1) Safe cleanup preview before execution

```bash
talpa clean --dry-run --json
```

Use this first in automation pipelines to estimate impact without mutating files.

### 2) Analyze large directories and focus candidates

```bash
talpa analyze ~/Downloads --depth 5 --min-size 104857600 --only-candidates --action inspect
```

This surfaces larger cleanup candidates (>=100 MiB / 104,857,600 bytes) with bounded scan depth.

### 3) Purge build artifacts across multiple workspaces

```bash
talpa purge --paths ~/Projects,~/Code --recent-days 14 --dry-run
```

Useful for reclaiming space while preserving recently modified artifacts.

### 4) High-risk apply flow with explicit confirmations

```bash
talpa uninstall --apply --yes --confirm HIGH-RISK
```

High-risk actions require explicit intent. Keep this pattern for non-interactive runs.

## Troubleshooting

### Interactive mode does not open

- Ensure the command runs in a real TTY (not piped/non-interactive shell).
- Ensure `TERM` is set and not `dumb`.
- If needed, run explicit command mode (e.g. `talpa status`).

### Command rejected with safety/path errors

- Re-run with `--dry-run --debug` to inspect validation behavior.
- Confirm target path is not in blocked critical paths.
- Check symlink targets stay within expected scope.

### No operation log file found

- Default path: `~/.config/talpa/operations.log`.
- If `XDG_CONFIG_HOME` is set, the log path becomes `$XDG_CONFIG_HOME/talpa/operations.log`.
- Verify you are not using `--no-oplog` and `TALPA_NO_OPLOG` is not set to `1`.

### JSON output changed unexpectedly in scripts

- Check `schema_version` in output.
- Validate against [`docs/JSON_SCHEMA.md`](docs/JSON_SCHEMA.md).
- Prefer resilient parsers and avoid brittle positional assumptions.

## Compatibility

- Primary target: Linux distributions
  - Ubuntu/Debian family
  - Fedora/RHEL family
  - Arch Linux
  - openSUSE
- Release architectures: `amd64`, `arm64`

See compatibility notes:

- [`docs/COMPATIBILITY_MATRIX.md`](docs/COMPATIBILITY_MATRIX.md)

## Development

Common local checks (aligned with CI):

```bash
go test ./...
go vet ./...
go test -race ./...
bash scripts/ci/smoke_json.sh
bash scripts/ci/smoke_negative.sh
bash scripts/ci/check_json_schema_sync.sh
```

Release workflow builds:

```bash
GOOS=linux GOARCH=amd64 go build -o dist/talpa-linux-amd64 ./cmd/talpa
GOOS=linux GOARCH=arm64 go build -o dist/talpa-linux-arm64 ./cmd/talpa
sha256sum dist/talpa-linux-amd64 dist/talpa-linux-arm64 > dist/SHA256SUMS
```

## Roadmap

- **v0.x (MVP):** safety core, clean/analyze/purge/status, completion/update/remove, JSON stability
- **v1.0:** stronger uninstall/installer/optimize workflows and package-manager integration depth
- **v1.x+:** ruleset plugins, profiles, optional launcher integrations

Detailed plan:

- [`docs/ROADMAP.md`](docs/ROADMAP.md)

## Contributing

Contributions are welcome. Please read:

- [`docs/CONTRIBUTING.md`](docs/CONTRIBUTING.md)
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)
- [`docs/TEST_STRATEGY.md`](docs/TEST_STRATEGY.md)

## Security

If you discover a vulnerability, please report it privately through GitHub Security Advisories.

- See [`SECURITY.md`](SECURITY.md) for full policy and response expectations.

## License

This project is licensed under the **MIT License**. See [LICENSE](LICENSE).

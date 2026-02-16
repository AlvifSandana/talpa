# Talpa

Talpa is a Linux-first single-binary CLI/TUI for deep cleanup, disk analysis, project artifact purge, and system health monitoring, designed with a strict safety-first model.

## Why Talpa
Linux maintenance workflows are often fragmented across many tools. Talpa unifies common cleanup and optimization operations into one consistent interface with strong safety controls.

## Key Features

### MVP (v0.x)
- `talpa clean` — safe cache/log/temp cleanup
- `talpa analyze` — interactive disk explorer
- `talpa purge` — remove project build artifacts
- `talpa status` — real-time system metrics
- `talpa completion`, `talpa update`, `talpa remove`

### v1.0
- `talpa uninstall` — uninstall app + XDG leftovers
- `talpa installer` — remove installer artifacts
- `talpa optimize` — safe cross-distro optimization actions
- package manager integration: apt/dnf/pacman/zypper + snap/flatpak

## Safety Principles
- Dry-run support: `--dry-run`
- Strict path validation and blocked critical paths
- Symlink traversal protection
- Layered confirmations for destructive actions
- Local JSONL operation audit logs
- No telemetry by default

## Planned Installation
Release installation script and binaries will be published in GitHub Releases.

Example (planned):
```bash
curl -fsSL https://<release-url>/install.sh | bash
```

Manual install target:
- User: `~/.local/bin/talpa`
- System: `/usr/local/bin/talpa`

## Usage

Interactive mode:
```bash
talpa
```

CLI mode:
```bash
talpa clean --dry-run
talpa analyze ~
talpa purge --paths ~/Projects,~/Code
talpa status
```

Global flags:
- `--dry-run`
- `--debug`
- `--yes`
- `--json`
- `--no-oplog`

## Compatibility Targets
- Ubuntu/Debian
- Fedora/RHEL-family
- Arch Linux
- openSUSE
- Architectures: amd64, arm64

## Roadmap
- v0.x (MVP): clean, analyze, purge, status + safety core
- v1.0: uninstall, installer, optimize + distro integrations
- v1.x+: plugin rulesets, profiles, optional launcher integrations

## Contributing
Contribution guidelines and architecture references will be available in:
- `docs/CONTRIBUTING.md`
- `docs/ARCHITECTURE.md`
- `docs/SAFETY_MODEL.md`
- `docs/TEST_STRATEGY.md`

Additional references:
- `docs/JSON_SCHEMA.md`
- `docs/OPERATIONS_LOG.md`

## License
License will be finalized before public release.

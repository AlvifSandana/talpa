# Contributing

## Getting Started
1. Install Go 1.22 (see `go.mod`).
2. Clone the repository.
3. Run tests locally before opening a PR.

## Development Workflow
- Keep changes scoped and focused.
- Follow safety-first design for any destructive action.
- Ensure `--dry-run` and `--json` outputs remain consistent.

## Testing Expectations
- Add unit tests for safety, rules, and scanner behavior.
- Update golden JSON tests if schema changes (with justification).
- Run integration tests with fixtures for filesystem behavior.

## Whitelist Format
- File location: `~/.config/talpa/whitelist` (or `$XDG_CONFIG_HOME/talpa/whitelist`)
- Supports:
  - exact path, e.g. `/usr/local/bin/talpa`
  - directory prefix entries
  - limited glob (`*`, `?`, `[]`) with filepath match semantics

## Pull Request Checklist
- Tests pass locally.
- Docs updated when changing user-visible behavior.
- Operation log behavior remains intact.
- No telemetry added without explicit approval.

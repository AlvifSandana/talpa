# Test Strategy

## Goals
- Verify safety guarantees (blocked paths, symlink guard, dry-run).
- Ensure scan performance and stability on large directories.
- Maintain output stability for `--json`.
- Validate cross-distro compatibility for supported commands.

## Unit Tests
- Path validator: null/control chars, traversal, blocked paths.
- Symlink guard: escape detection and loop prevention.
- Rule matching: clean/purge/installer/uninstall rules.
- Risk classification: low/medium/high correctness.
- Recent project detection (purge protection).

## Integration Tests
- Fixture-based home directory scans.
- Network mount skip behavior.
- Read-only filesystem behavior.
- Command E2E in dry-run mode.

## Golden JSON Tests
- Snapshot output for each command in `--json` mode.
- Schema version pinned (`schema_version`).
- Stable ordering of keys to reduce noise.

## Manual Checklist (Destructive Safety)
- `--dry-run` does not modify files.
- Blocked paths cannot be deleted.
- High-risk items require double confirm.
- `--yes` only works for scripted flows.
- Operation log entries are written for each action.

## CI Matrix
### Tier-1 (required)
- Ubuntu latest
- Fedora latest
- Arch latest

### Tier-2 (best effort)
- Debian stable
- openSUSE Leap/Tumbleweed

## CI Pipeline Outline
1. Lint
2. Unit tests
3. Integration tests (fixtures)
4. Golden JSON verification
5. Smoke tests for CLI commands

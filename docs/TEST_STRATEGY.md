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

### Tier-2 (best effort)
- Fedora latest
- Arch latest

## CI Pipeline Outline
1. Unit tests (`go test ./...`)
2. Static analysis (`go vet ./...`)
3. Concurrency checks (`go test -race ./...`)
4. CLI JSON smoke checks (all core commands in dry-run-safe mode)
5. CLI negative smoke checks (invalid flags/inputs fail with expected errors)
6. Cross-distro best-effort test run (Fedora, Arch)

## CI Policy
- Blocking gate: `test` job on Ubuntu must pass.
- Best-effort gate: `distro-best-effort` (Fedora/Arch) is non-blocking but must be investigated when failing.
- Workflow concurrency cancels older in-progress runs on the same ref.
- Job and container-level timeouts are enabled to avoid stuck pipelines.
- Dependency install in distro jobs uses retry with backoff for transient mirror/network failures.
- On failure, CI uploads step logs as artifacts for faster diagnosis.

## Local CI Parity
- Use `scripts/ci/run_local_ci.sh` to run the same core test commands as the CI `test` job:
  1. `go test ./...`
  2. `go vet ./...`
  3. `go test -race ./...`
  4. `scripts/ci/smoke_json.sh`
  5. `scripts/ci/smoke_negative.sh`

Future expansion target for Tier-2: Debian stable and openSUSE Leap/Tumbleweed.

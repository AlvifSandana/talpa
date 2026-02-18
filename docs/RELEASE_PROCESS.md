# Release Process

## Build Targets
- Linux amd64
- Linux arm64

## Release Steps
1. Run full test suite (unit + integration + golden JSON).
2. Run schema sync gate (`scripts/ci/check_json_schema_sync.sh`).
2. Build binaries for target architectures.
3. Generate SHA256 checksums for each artifact.
4. Publish GitHub Release with release notes and checksums.
5. Update installer script to point to latest release.

Reference workflows/scripts:
- `.github/workflows/release.yml`
- `scripts/ci/release_dry_run.sh`

## CI Container Digest Rotation
Best-effort distro CI jobs use pinned container image digests for reproducibility.

Rotation checklist:
1. Pull latest tag locally (`docker pull fedora:41` and `docker pull archlinux:latest`).
2. Resolve digest (`docker inspect --format='{{index .RepoDigests 0}}' <image:tag>`).
3. Update pinned values in `.github/workflows/ci.yml`.
4. Run local smoke checks (`scripts/ci/smoke_json.sh` and `scripts/ci/smoke_negative.sh`).
5. Open PR with a short note describing which digests were rotated.

Recommended cadence: monthly, or immediately for high-severity base image CVEs.

CI note:
- Steps that pipe output to `tee` must use `set -o pipefail` so upstream command failures are not masked.
- CI uploads collected logs as artifacts when jobs fail to speed up troubleshooting.

## CI Troubleshooting
When CI fails:
1. Open the failed job and download the uploaded artifact (`ci-logs-test` or `ci-logs-distro-*`).
2. Review step logs in order (`go-test.log`, `go-vet.log`, `go-test-race.log`, `smoke-json.log`, `smoke-negative.log`).
3. For distro failures, inspect `distro-<name>.log` first to separate package install failures from Go test failures.
4. Reproduce locally with the same scripts (`scripts/ci/smoke_json.sh`, `scripts/ci/smoke_negative.sh`) and test commands.

Operational notes:
- CI uses per-job timeouts and container execution timeout guards to avoid stuck runs.
- Distro dependency install steps use retry with backoff for transient network failures.
- `distro-best-effort` is non-blocking (`continue-on-error: true`), so investigate failures even when the overall workflow is green.

## Verification
- Installer script must verify SHA256.
- All downloads must be via HTTPS.
- Release dry-run must pass checksum verification locally before publish.

## Temporary Security Exception (Current Cycle)
- Scope: `analyze --action trash` path safety hardening is significantly improved, but not yet at full adversarial TOCTOU-proof level.
- Platform behavior: Unix uses fd-based/no-follow flow; non-Unix trash action uses pragmatic rename fallback with reduced safety guarantees.
- Risk acceptance: approved for Linux-first/local trusted-user usage in this cycle.
- Required release note line: mention limitation explicitly and mark deeper hardening as next-cycle follow-up.
- Exit criteria to remove exception: security gate passes strict TOCTOU review without caveat.

## Rollback
- If release is broken, publish a patch release and update installer to latest stable.
- Document known issues in release notes.

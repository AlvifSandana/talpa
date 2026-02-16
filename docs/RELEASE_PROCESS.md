# Release Process

## Build Targets
- Linux amd64
- Linux arm64

## Release Steps
1. Run full test suite (unit + integration + golden JSON).
2. Build binaries for target architectures.
3. Generate SHA256 checksums for each artifact.
4. Publish GitHub Release with release notes and checksums.
5. Update installer script to point to latest release.

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

## Verification
- Installer script must verify SHA256.
- All downloads must be via HTTPS.

## Rollback
- If release is broken, publish a patch release and update installer to latest stable.
- Document known issues in release notes.

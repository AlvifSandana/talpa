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

## Verification
- Installer script must verify SHA256.
- All downloads must be via HTTPS.

## Rollback
- If release is broken, publish a patch release and update installer to latest stable.
- Document known issues in release notes.

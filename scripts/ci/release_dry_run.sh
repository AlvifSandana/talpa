#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
cd "$ROOT_DIR"

mkdir -p dist

echo "[release-dry-run] build linux amd64"
GOOS=linux GOARCH=amd64 go build -o dist/talpa-linux-amd64 ./cmd/talpa

echo "[release-dry-run] build linux arm64"
GOOS=linux GOARCH=arm64 go build -o dist/talpa-linux-arm64 ./cmd/talpa

echo "[release-dry-run] generate checksums"
sha256sum dist/talpa-linux-amd64 dist/talpa-linux-arm64 > dist/SHA256SUMS

echo "[release-dry-run] verify checksums"
sha256sum -c dist/SHA256SUMS

echo "[release-dry-run] success"

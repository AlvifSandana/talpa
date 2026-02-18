#!/usr/bin/env bash

set -euo pipefail

# Runs local equivalent of CI test-job core checks.

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
cd "$ROOT_DIR"

command -v go >/dev/null 2>&1 || { echo "[local-ci] go is required"; exit 1; }
command -v bash >/dev/null 2>&1 || { echo "[local-ci] bash is required"; exit 1; }

echo "[local-ci] go test"
go test ./...

echo "[local-ci] go vet"
go vet ./...

echo "[local-ci] go test -race"
go test -race ./...

echo "[local-ci] smoke json"
bash scripts/ci/smoke_json.sh

echo "[local-ci] smoke negative"
bash scripts/ci/smoke_negative.sh

echo "[local-ci] schema sync"
bash scripts/ci/check_json_schema_sync.sh

echo "[local-ci] completed"

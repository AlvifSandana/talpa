#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
cd "$ROOT_DIR"

BIN_PATH="$(mktemp /tmp/talpa-smoke.XXXXXX)"
trap 'rm -f "$BIN_PATH"' EXIT
go build -o "$BIN_PATH" ./cmd/talpa

run_and_validate_json() {
  local name="$1"
  shift

  echo "[smoke] $name"
  TALPA_NO_OPLOG=1 timeout 30s "$BIN_PATH" "$@" | python3 -m json.tool >/dev/null
}

run_and_validate_json "clean" clean --json --dry-run
run_and_validate_json "analyze" analyze . --json --dry-run --depth 1 --limit 5
run_and_validate_json "purge" purge --json --dry-run --paths .
run_and_validate_json "status" status --json --top 1 --interval 1
run_and_validate_json "update" update --json --dry-run
run_and_validate_json "remove" remove --json --dry-run
run_and_validate_json "uninstall" uninstall --json --dry-run
run_and_validate_json "installer" installer --json --dry-run
run_and_validate_json "optimize" optimize --json --dry-run

echo "[smoke] all JSON command checks passed"

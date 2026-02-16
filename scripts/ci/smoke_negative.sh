#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
cd "$ROOT_DIR"

BIN_PATH="$(mktemp /tmp/talpa-smoke-negative.XXXXXX)"
trap 'rm -f "$BIN_PATH"' EXIT
go build -o "$BIN_PATH" ./cmd/talpa

assert_fail_contains() {
  local name="$1"
  local expected="$2"
  shift 2

  local output
  local rc
  set +e
  output="$(TALPA_NO_OPLOG=1 timeout 30s "$BIN_PATH" "$@" 2>&1)"
  rc=$?
  set -e
  if [ "$rc" -eq 0 ]; then
    echo "[smoke-negative] expected failure for '$name' but command succeeded"
    exit 1
  fi

  case "$output" in
    *"$expected"*)
      echo "[smoke-negative] $name"
      ;;
    *)
      echo "[smoke-negative] expected output for '$name' to contain: $expected"
      echo "$output"
      exit 1
      ;;
  esac
}

assert_fail_contains "status invalid interval" "--interval must be >= 1" status --interval 0
assert_fail_contains "status invalid top" "--top must be >= 1" status --top 0
assert_fail_contains "analyze invalid depth" "--depth must be >= 1" analyze --depth 0
assert_fail_contains "analyze invalid sort" "--sort must be one of" analyze --sort invalid
assert_fail_contains "analyze invalid min-size" "--min-size must be >= 0" analyze --min-size -1
assert_fail_contains "uninstall invalid target" "invalid target" uninstall --dry-run --target invalid
assert_fail_contains "installer apply requires confirm" "confirmation required for installer cleanup" installer --apply
assert_fail_contains "optimize apply requires confirm" "confirmation required for optimize" optimize --apply

echo "[smoke-negative] all negative command checks passed"

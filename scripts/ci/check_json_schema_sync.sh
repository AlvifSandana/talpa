#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"
cd "$ROOT_DIR"

SCHEMA_FILE="docs/JSON_SCHEMA.md"

require_in_schema() {
  local expected="$1"
  if ! grep -Fq -- "$expected" "$SCHEMA_FILE"; then
    echo "[schema-sync] missing in docs/JSON_SCHEMA.md: $expected"
    exit 1
  fi
}

echo "[schema-sync] checking command coverage"
# use explicit heading tokens to avoid markdown heading variations
require_in_schema "### \`clean\`"
require_in_schema "### \`analyze\`"
require_in_schema "### \`purge\`"
require_in_schema "### \`status\`"
require_in_schema "### \`optimize\`"
require_in_schema "### \`uninstall\`"
require_in_schema "### \`installer\`"
require_in_schema "### \`update\`"
require_in_schema "### \`remove\`"

echo "[schema-sync] checking status metrics fields"
require_in_schema "memory_total_bytes"
require_in_schema "memory_used_bytes"
require_in_schema "swap_used_bytes"
require_in_schema "swap_total_bytes"
require_in_schema "disk_io"
require_in_schema "ip_addresses"

echo "[schema-sync] checking analyze result/action notes"
require_in_schema "inspect"
require_in_schema "candidate"
require_in_schema "trashed"
require_in_schema "--action inspect|trash|delete"

echo "[schema-sync] schema docs look synchronized"

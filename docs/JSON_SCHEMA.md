# JSON Output Schema

This document defines the JSON output structure for Talpa commands using the `--json` flag.

## Versioning
- Schema is versioned using `schema_version`.
- Backward-incompatible changes require a major version bump.

## Common Envelope
All commands return a common envelope:

```json
{
  "schema_version": "1.0",
  "command": "clean",
  "timestamp": "2026-02-16T10:00:00Z",
  "duration_ms": 1200,
  "dry_run": true,
  "summary": {
    "items_total": 42,
    "items_selected": 30,
    "estimated_freed_bytes": 123456789,
    "errors": 0
  },
  "items": [
    {
      "id": "item-1",
      "rule_id": "clean.xdg.cache",
      "path": "/home/user/.cache/app",
      "size_bytes": 1024,
      "last_modified": "2026-02-15T12:00:00Z",
      "category": "xdg_cache",
      "risk": "low",
      "selected": true,
      "requires_root": false,
      "result": "planned"
    }
  ]
}
```

## Field Definitions

### Envelope
- `schema_version`: string, required.
- `command`: string, required. One of: `clean`, `analyze`, `purge`, `status`, `optimize`, `uninstall`, `installer`, `update`, `remove`.
- `timestamp`: RFC3339 time, required.
- `duration_ms`: integer, optional.
- `dry_run`: boolean, required for destructive commands.
- `summary`: object, required.

### Summary
- `items_total`: integer.
- `items_selected`: integer.
- `estimated_freed_bytes`: integer.
- `errors`: integer. Includes execution failures and operation-log write failures.

### Item
- `id`: string.
- `rule_id`: string.
- `path`: string.
- `size_bytes`: integer.
- `last_modified`: RFC3339 time or null.
- `category`: string.
- `risk`: `low|medium|high`.
- `selected`: boolean.
- `requires_root`: boolean.
- `result`: string. Current values include `planned`, `already-skipped`, `skipped`, `inspect`, `candidate`, `trashed`, `deleted`, `updated`, `optimized`, `uninstalled`, and `error`.

## Command Notes

### `status`
Returns system metrics instead of item list:

```json
{
  "schema_version": "1.0",
  "command": "status",
  "timestamp": "2026-02-16T10:00:00Z",
  "duration_ms": 1000,
  "metrics": {
    "cpu_usage": 0.12,
    "load_avg": [0.5, 0.7, 0.9],
    "memory_total_bytes": 17179869184,
    "memory_used_bytes": 123456789,
    "swap_used_bytes": 0,
    "swap_total_bytes": 0,
    "disk_usage": [
      {"mount": "/", "used_bytes": 123, "total_bytes": 456}
    ],
    "disk_io": {"read_bytes": 1000, "write_bytes": 2000, "read_bps": 100, "write_bps": 200},
    "net": {"tx_bytes": 1, "rx_bytes": 2, "tx_bps": 10, "rx_bps": 20},
    "ip_addresses": ["192.168.1.10"]
  }
}
```

### `clean`
```json
{
  "schema_version": "1.0",
  "command": "clean",
  "timestamp": "2026-02-16T10:00:00Z",
  "duration_ms": 850,
  "dry_run": true,
  "summary": {
    "items_total": 12,
    "items_selected": 9,
    "estimated_freed_bytes": 98765432,
    "errors": 0
  },
  "items": [
    {
      "id": "item-1",
      "rule_id": "clean.browser.cache",
      "path": "/home/user/.cache/chromium",
      "size_bytes": 4096,
      "last_modified": "2026-02-15T12:00:00Z",
      "category": "browser_cache",
      "risk": "low",
      "selected": true,
      "requires_root": false,
      "result": "planned"
    }
  ]
}
```

### `analyze`
- `items` represent tree nodes with aggregated sizes.
- `result` defaults to `inspect|candidate` and can become `trashed|deleted|skipped|error` when using action mode.
- Use `--action inspect|trash|delete` to control analyze action flow.

Example:
```json
{
  "schema_version": "1.0",
  "command": "analyze",
  "timestamp": "2026-02-16T10:00:00Z",
  "duration_ms": 5000,
  "dry_run": false,
  "summary": {
    "items_total": 4,
    "items_selected": 1,
    "estimated_freed_bytes": 12345,
    "errors": 0
  },
  "items": [
    {
      "id": "node-1",
      "rule_id": "",
      "path": "/home/user/Downloads",
      "size_bytes": 12345,
      "last_modified": "2026-02-10T10:00:00Z",
      "category": "tree_node",
      "risk": "medium",
      "selected": false,
      "requires_root": false,
      "result": "planned"
    }
  ]
}
```

### `purge`
```json
{
  "schema_version": "1.0",
  "command": "purge",
  "timestamp": "2026-02-16T10:00:00Z",
  "duration_ms": 1400,
  "dry_run": true,
  "summary": {
    "items_total": 6,
    "items_selected": 2,
    "estimated_freed_bytes": 543210,
    "errors": 0
  },
  "items": [
    {
      "id": "item-2",
      "rule_id": "purge.node_modules",
      "path": "/home/user/Projects/app/node_modules",
      "size_bytes": 543210,
      "last_modified": "2026-02-01T09:00:00Z",
      "category": "project_artifact",
      "risk": "low",
      "selected": true,
      "requires_root": false,
      "result": "planned"
    }
  ]
}
```

### `optimize`
```json
{
  "schema_version": "1.0",
  "command": "optimize",
  "timestamp": "2026-02-16T10:00:00Z",
  "duration_ms": 900,
  "dry_run": true,
  "summary": {
    "items_total": 3,
    "items_selected": 3,
    "estimated_freed_bytes": 0,
    "errors": 0
  },
  "items": [
    {
      "id": "action-1",
      "rule_id": "optimize.journal.vacuum",
      "path": "systemd-journal",
      "size_bytes": 0,
      "last_modified": null,
      "category": "system_maintenance",
      "risk": "medium",
      "selected": true,
      "requires_root": true,
      "result": "planned"
    }
  ]
}
```

### `uninstall`
```json
{
  "schema_version": "1.0",
  "command": "uninstall",
  "timestamp": "2026-02-16T10:00:00Z",
  "duration_ms": 2200,
  "dry_run": true,
  "summary": {
    "items_total": 2,
    "items_selected": 2,
    "estimated_freed_bytes": 20480,
    "errors": 0
  },
  "items": [
    {
      "id": "app-1",
      "rule_id": "uninstall.leftover",
      "path": "/home/user/.config/app",
      "size_bytes": 20480,
      "last_modified": "2026-01-20T10:00:00Z",
      "category": "leftover",
      "risk": "medium",
      "selected": true,
      "requires_root": false,
      "result": "planned"
    }
  ]
}
```

### `installer`
```json
{
  "schema_version": "1.0",
  "command": "installer",
  "timestamp": "2026-02-16T10:00:00Z",
  "duration_ms": 600,
  "dry_run": true,
  "summary": {
    "items_total": 1,
    "items_selected": 1,
    "estimated_freed_bytes": 1234567,
    "errors": 0
  },
  "items": [
    {
      "id": "inst-1",
      "rule_id": "installer.appimage",
      "path": "/home/user/Downloads/app.AppImage",
      "size_bytes": 1234567,
      "last_modified": "2026-02-01T10:00:00Z",
      "category": "installer",
      "risk": "low",
      "selected": true,
      "requires_root": false,
      "result": "planned"
    }
  ]
}
```

### `update`
```json
{
  "schema_version": "1.0",
  "command": "update",
  "timestamp": "2026-02-16T10:00:00Z",
  "duration_ms": 1200,
  "dry_run": false,
  "summary": {
    "items_total": 1,
    "items_selected": 1,
    "estimated_freed_bytes": 0,
    "errors": 0
  },
  "items": [
    {
      "id": "update-1",
      "rule_id": "update.binary",
      "path": "/usr/local/bin/talpa",
      "size_bytes": 0,
      "last_modified": null,
      "category": "self_update",
      "risk": "medium",
      "selected": true,
      "requires_root": true,
      "result": "planned"
    }
  ]
}
```

### `remove`
```json
{
  "schema_version": "1.0",
  "command": "remove",
  "timestamp": "2026-02-16T10:00:00Z",
  "duration_ms": 400,
  "dry_run": false,
  "summary": {
    "items_total": 1,
    "items_selected": 1,
    "estimated_freed_bytes": 0,
    "errors": 0
  },
  "items": [
    {
      "id": "remove-1",
      "rule_id": "remove.binary",
      "path": "/usr/local/bin/talpa",
      "size_bytes": 0,
      "last_modified": null,
      "category": "self_remove",
      "risk": "high",
      "selected": true,
      "requires_root": true,
      "result": "planned"
    }
  ]
}
```

## Stability Guarantees
- Field names remain stable within a major version.
- Optional fields may be added in minor versions.

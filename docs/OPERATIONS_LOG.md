# Operations Log

Talpa writes an audit trail of all file and command operations to:

```
~/.config/talpa/operations.log
```

The log uses JSON Lines (one JSON object per line).

## Log Entry Fields
- `timestamp`: RFC3339 time.
- `plan_id`: identifier for the operation plan.
- `command`: the executed command.
- `action`: operation type. Current values include `delete`, `exec`, and `skip`.
- `path`: target path (when applicable).
- `rule_id`: rule identifier (when applicable).
- `category`: rule category.
- `size_bytes`: size of target (if known).
- `risk`: `low|medium|high`.
- `result`: operation outcome. Current values include `planned`, `already-skipped`, `skipped`, `deleted`, `updated`, `optimized`, `uninstalled`, and `error`.
- `error`: error message (if any).
- `duration_ms`: execution time (if applicable).
- `dry_run`: boolean.
- `user_id`: numeric uid (if available).

## Example Entry

```json
{
  "timestamp": "2026-02-16T10:00:00Z",
  "plan_id": "plan-123",
  "command": "clean",
  "action": "delete",
  "path": "/home/user/.cache/app",
  "rule_id": "clean.xdg.cache",
  "category": "xdg_cache",
  "size_bytes": 1048576,
  "risk": "low",
  "result": "deleted",
  "error": "",
  "duration_ms": 12,
  "dry_run": false,
  "user_id": 1000
}
```

## Disabling the Log
The log can be disabled with:
- `--no-oplog` flag
- `TALPA_NO_OPLOG=1` environment variable

## Privacy
- No file contents are recorded.
- Logs remain local by default.

## Error Taxonomy
Use a consistent `error_code` prefix in `error` details when possible.

Suggested codes:
- `PATH_BLOCKED`: target path is in blocked list.
- `PATH_INVALID`: invalid path (null/control/traversal).
- `SYMLINK_ESCAPE`: symlink resolved outside allowed scope.
- `PERMISSION_DENIED`: insufficient privileges.
- `NOT_FOUND`: target path missing at execution time.
- `IO_ERROR`: filesystem I/O error.
- `TIMEOUT`: external command timed out.
- `CMD_FAILED`: external command returned non-zero.
- `RULE_SKIPPED`: rule disabled or filtered.
- `USER_CANCELLED`: user declined confirmation.

Example with error code:

```json
{
  "timestamp": "2026-02-16T10:01:00Z",
  "plan_id": "plan-124",
  "command": "clean",
  "action": "delete",
  "path": "/etc/passwd",
  "rule_id": "clean.unsafe",
  "category": "system",
  "size_bytes": 0,
  "risk": "high",
  "result": "error",
  "error": "PATH_BLOCKED: /etc",
  "duration_ms": 1,
  "dry_run": false,
  "user_id": 1000
}
```

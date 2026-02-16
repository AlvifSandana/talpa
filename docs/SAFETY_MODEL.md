# Safety Model

## Principles
Talpa uses safety-first defaults to avoid accidental destructive actions:
- No telemetry by default.
- Destructive operations require explicit confirmation.
- All file operations are validated by a centralized safety gate.

## Safety Gate
Every delete/modify operation must pass:
1. Path validation
2. Blocked path checks
3. Symlink guard and scope enforcement
4. Privilege boundary checks

## Path Validation
Reject paths that are:
- Empty
- Contain null bytes or control characters
- Contain traversal (`/../`)
- Not clean after canonicalization

## Blocked Paths
Hard block critical system paths:
- `/`
- `/boot`
- `/bin`, `/sbin`
- `/lib`, `/lib64`
- `/usr`
- `/etc`
- `/proc`, `/sys`, `/dev`, `/run`

`/var` is blocked by default except for explicit cache allowlists.

## Symlink Guard
- Default `followSymlink = false`.
- If a symlink is encountered, resolve target and validate it.
- Reject symlinks that escape allowed scope.
- Detect loops via inode/device tracking.

## Privilege Boundary
- Root operations only when required.
- Use least privilege and narrow scope to specific targets.
- Avoid global escalation for the entire command.

## External Command Safety
- All external commands must run with a timeout.
- Timeouts should terminate process trees.
- Errors are logged with minimal output (no sensitive data).

## Confirmation Policy
- `--dry-run` is supported for all destructive commands.
- High-risk items require double confirmation.
- `--yes` is allowed only for explicit non-interactive scripted usage.

## Operation Log
- JSONL log per action at `~/.config/talpa/operations.log`.
- Logs contain path, size, action, outcome, and error reason.
- No file contents are recorded.

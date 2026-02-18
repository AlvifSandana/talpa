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
- `--yes` is allowed for explicit non-interactive scripted usage.
- Double-confirm token uses `--confirm HIGH-RISK` for high-risk apply flows (`remove`, `uninstall --apply`, `installer --apply`, `optimize --apply`, and high-risk `analyze --action delete`).

## Whitelist Behavior
- Whitelist source: `~/.config/talpa/whitelist` (or `$XDG_CONFIG_HOME/talpa/whitelist`).
- Supported patterns:
  - exact path (`/usr/local/bin/talpa`)
  - path prefix (directory entries under whitelisted root)
  - limited glob pattern (`*`, `?`, `[]`) using filepath match semantics.

## Known Limitations (Temporary)
- `analyze --action trash` uses stronger fd-based safety controls on Unix platforms.
- On non-Unix platforms, `analyze --action trash` uses pragmatic rename fallback with reduced guarantees versus Unix fd-based safety semantics.
- A residual race window can still exist under adversarial local filesystem mutation (TOCTOU class) because path validation and kernel path resolution cannot be fully bound atomically in the current implementation.

Current threat-model stance for release:
- Accepted for Linux-first/local CLI usage with trusted local user context.
- Not treated as hardened against hostile same-host race attackers yet.
- Further hardening (next phase) targets stricter path-resolution guarantees.

## Operation Log
- JSONL log per action at `~/.config/talpa/operations.log`.
- Logs contain path, size, action, outcome, and error reason.
- No file contents are recorded.

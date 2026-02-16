# Ruleset Reference

This document describes built-in rule categories used for matching cleanup candidates.

## Clean Rules (User-Level)
- XDG caches: `~/.cache/*`
- Browser caches: Chromium/Chrome/Brave/Firefox
- Electron app caches: Slack/Discord/VSCode (via XDG)
- Dev tool caches:
  - Node: npm/yarn/pnpm cache
  - Python: pip/poetry cache, `__pycache__`
  - Rust: cargo registry cache
  - Java: gradle caches, maven `.m2/repository` (optional)
  - Go: `GOCACHE`, `GOMODCACHE`
- Trash: `~/.local/share/Trash/*`
- Thumbnails: `~/.cache/thumbnails`
- User logs: `~/.local/state` and app logs in `~/.local/share`

## Clean Rules (System-Level, Opt-in)
- Package manager cache: apt/dnf/pacman/zypper
- Journal vacuum (systemd-journald)
- `/tmp` and `/var/tmp` with safety rules

## Purge Rules (Project Artifacts)
- Node: `node_modules`, `.next`, `dist`, `build`
- Rust: `target`
- Java: `target`, `build`, `.gradle`
- Python: `.venv`, `venv`, `__pycache__`, `.pytest_cache`
- Go: `vendor` (optional), `bin` (optional)
- Mobile/others: `.dart_tool`, `Pods`, `DerivedData`

## Installer Rules
- `.deb`, `.rpm`, `.pkg.tar.*`, `.AppImage`, `.run`
- `.zip`, `.tar.gz`, `.tar.xz` matching installer heuristics
- Default locations: `~/Downloads`, `~/Desktop`

## Uninstall Leftover Rules
Leftovers are matched in:
- `~/.config/<app>`
- `~/.local/share/<app>`
- `~/.cache/<app>`
- `~/.local/state/<app>`

Matching strategy:
1. exact match
2. normalized match (lowercase, dash/underscore)
3. curated aliases (manual list)

## Risk Level Guidelines
- Low: caches, temp files, build artifacts
- Medium: large app data, logs
- High: unknown paths, user data with ambiguous ownership

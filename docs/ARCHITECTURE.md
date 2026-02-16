# Architecture

## Overview
Talpa uses a clean-ish layered architecture to keep domain rules and safety logic isolated from OS integrations.

```
cmd -> app services -> domain core -> infra adapters
                   \-> tui
```

## Components

### Command Layer
- CLI command tree and flag parsing.
- Input validation and output formatting (`human` or `json`).
- Invokes app services with a normalized context.

### App Services
- Orchestrates use-cases: clean, analyze, purge, status, optimize, uninstall, installer, update, remove.
- Builds `OperationPlan` from rules and scan results.
- Applies dry-run and confirmation policy.

### Domain Core
- Rule engine (matching + risk classification).
- Safety policy (path validation, blocked paths, symlink guard).
- Domain models and operation planning.

### Infrastructure Adapters
- Filesystem scanner (mount-aware, loop-safe, depth-limited).
- Package manager adapters (apt/dnf/pacman/zypper/snap/flatpak).
- Metrics provider (gopsutil + /proc fallback).
- External command runner with timeouts.
- Config store and operation log writer.

### TUI
- Bubble Tea based views for interactive selection.
- Shared components for confirm dialogs and list filters.

## Package Layout (Recommended)

```
cmd/
  talpa/
    main.go
  root.go
  clean.go
  analyze.go
  purge.go
  status.go
  optimize.go
  uninstall.go
  installer.go
  update.go
  remove.go
  completion.go

internal/
  app/
    clean/service.go
    analyze/service.go
    purge/service.go
    status/service.go
    optimize/service.go
    uninstall/service.go
    installer/service.go
    update/service.go
    remove/service.go
    common/planner.go
    common/executor.go

  domain/
    model/
      rule.go
      candidate_item.go
      operation_plan.go
      operation_log_entry.go
      scan_context.go
      safety_policy.go
    rules/
      clean_rules.go
      purge_rules.go
      installer_rules.go
      uninstall_leftover_rules.go
    safety/
      path_validator.go
      symlink_guard.go
      blocked_paths.go
      risk_assessor.go
    ports/
      filesystem.go
      process.go
      pkgmanager.go
      metrics.go
      command_runner.go
      config_store.go
      oplog.go
      updater.go
      clock.go

  infra/
    filesystem/
      walker.go
      stat.go
      trash.go
    procfs/
      metrics_linux.go
    pkgmgr/
      detector.go
      apt.go
      dnf.go
      pacman.go
      zypper.go
      snap.go
      flatpak.go
      nullpkg.go
    exec/
      runner.go
    logging/
      logger.go
      oplog_jsonl.go
    config/
      store.go
      settings.go
    update/
      github_release.go
      checksum.go
    system/
      privilege.go
      mounts.go
      battery.go
      readonly_fs.go

tui/
  app.go
  pages/
    menu.go
    clean.go
    analyze.go
    purge.go
    status.go
  components/
    table.go
    confirm.go
    filter.go
  state/
    model.go
```

## Execution Flow (High-Level)
1. Load config and safety policy.
2. Build scan context and rule set for the command.
3. Scan and collect candidate items.
4. Apply safety gates and risk classification.
5. Present plan (TUI/CLI) and confirm.
6. Execute operations with bounded concurrency and timeouts.
7. Write JSONL operation logs and summary.

## Core Models (Summary)
- `Rule`: matching pattern, category, risk, default selection.
- `CandidateItem`: path + size + risk + selection state.
- `OperationPlan`: command + items + dry-run + estimates.
- `SafetyPolicy`: blocked paths, whitelist, symlink guard, timeouts.

## Architecture Decisions (Summary)
- Cobra for CLI.
- Bubble Tea + Lip Gloss for TUI.
- Centralized safety gate for destructive actions.
- JSONL operation log by default.
- Concurrent scanner with bounded workers.
- Package manager abstraction with capability detection.

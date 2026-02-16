# Implementation Plan — Talpa (Linux Universal)

## 1. Executive Overview
Talpa adalah single-binary CLI/TUI Linux untuk clean, analyze, purge, status, dan (v1.0) uninstall/optimize/installer cleanup dengan prinsip safety-first.

### Product Goals
1. Reclaim disk space secara aman dan terukur.
2. Menyatukan workflow maintenance Linux ke satu tool.
3. Menjamin kompatibilitas lintas distro mayor.
4. Menjaga keamanan aksi destruktif via guardrails ketat.

### MVP Scope (v0.x)
- `clean`, `analyze`, `purge`, `status`
- `completion`, `update`, `remove`
- Safety core: `--dry-run`, whitelist, blocked paths, operation log

### v1.0 Scope
- `uninstall`, `installer`, `optimize`
- Integrasi apt/dnf/pacman/zypper + snap/flatpak

## 2. Architecture & Implementation Strategy

### Recommended Stack
- Language: Go
- CLI: Cobra (recommended)
- TUI: Bubble Tea + Lip Gloss
- Metrics: gopsutil + `/proc` fallback
- Logs: JSONL operation log

### Target Architecture
`cmd -> app services -> domain core -> infra adapters`

- Command layer: parsing command/flags, output mode (human/json)
- App services: orchestration use-case + operation planning
- Domain: rule engine, risk model, safety policy
- Infra: filesystem scanner, package manager adapters, procfs metrics, command runner timeout, config/log store

### Core Data Models
- `Rule`
- `CandidateItem`
- `RiskLevel` (`low|medium|high`)
- `OperationPlan`
- `OperationLogEntry`
- `ScanContext`
- `SafetyPolicy`

## 3. Milestones

| Milestone | Objective | Deliverables | Estimate |
|---|---|---|---|
| M0 Foundation | Baseline project | repo structure, Cobra skeleton, config loading, CI basic | 1 week |
| M1 Safety + Scanner Core | Fondasi keamanan & performa | path validator, symlink guard, blocked paths, concurrent scanner, timeout wrapper | 2 weeks |
| M2 MVP Commands | Core user value | `clean`, `analyze`, `purge`, `status`, JSON output, operation log, TUI menu basic | 4 weeks |
| M3 MVP Hardening | Kualitas rilis MVP | `update`, `remove`, installer script+checksum, QA suite, docs user | 2 weeks |
| M4 v1.0 Expansion | Fitur lanjutan | `uninstall`, `installer`, `optimize`, package manager adapters | 4 weeks |
| M5 Stabilization | Reliability & release | perf tuning, bugfix, distro validation, release notes | 1-2 weeks |

## 4. Work Breakdown Structure (WBS)

### A. CLI/TUI
- Command tree dan global flags konsisten
- Interactive selection, filtering, confirmation dialog
- `--json` parity lintas command

### B. Scanning Engine
- Concurrent directory walk
- Depth control, excludes, mount-aware skip
- Loop-safe symlink behavior
- Size aggregation + age buckets

### C. Rules Engine
- Built-in rules untuk MVP
- Risk classification
- Whitelist matching (exact + limited glob)
- Leftover detection strategy (exact > normalized > alias)

### D. Safety & Security
- SR-1 path validation
- Hard blocked paths
- Privilege boundary (sudo minimization)
- External command timeout + failure isolation
- No telemetry default

### E. Distro Integration
- Package manager abstraction + capability detection
- Graceful degradation jika backend unavailable

### F. Observability
- `operations.log` JSONL
- Debug mode (`--debug`) dengan alasan match/reject
- Internal counters: scan duration, items found, estimated reclaimed bytes

### G. Packaging & Distribution
- Cross-compile linux amd64/arm64
- Installer script + SHA256 verification
- Self-update with checksum validation

### H. QA & Testing
- Unit: validator/rules/scanner/safety
- Integration: fixture-based filesystem tests
- Golden tests untuk schema `--json`
- CI distro matrix

## 5. Prioritized Backlog

### P0 (must-have for MVP)
- Safety core lengkap
- Scanner core performant + stable
- `clean`, `analyze`, `purge`, `status`
- Operation log + JSON output
- CI basic + golden tests
- `completion`, `update`, `remove`

### P1 (v1.0)
- `uninstall` + leftover cleanup
- `installer` cleanup
- `optimize` safe actions + preflight guard
- Full distro adapter coverage

### P2 (v1.x+)
- Plugin ruleset (YAML/JSON)
- Profiles (Dev/Desktop/Minimal)
- Optional launcher integrations

## 6. Testing Strategy

### Unit Tests
- path traversal/null/control chars rejection
- blocked path handling
- symlink escape prevention
- rule matching + risk scoring
- recent-project detector

### Integration Tests
- synthetic home fixture (cache/log/build artifacts)
- symlink loop scenarios
- read-only path and timeout behavior
- command E2E in dry-run mode

### Golden JSON
- stable schema snapshots
- schema versioning (`schema_version`)

### CI Matrix
- Tier-1: Ubuntu, Fedora, Arch
- Tier-2: Debian, openSUSE (best effort)

## 7. Risk Register

| Risk | Impact | Mitigation |
|---|---|---|
| False positive deletion | Critical | strict safety gate, default low/med selection, double confirm for high |
| Slow scan on large home dirs | High | bounded worker pool, excludes, depth limit, network mount skip |
| Cross-distro command differences | High | adapter abstraction + capability checks + fallback |
| Root operation hazards | High | minimal privilege, preflight checks, scoped actions only |

## 8. Architecture Decisions (ADR Mini)

1. Use Cobra for command surface — Accepted
2. Use Bubble Tea + Lip Gloss for TUI — Accepted
3. Centralized safety gate before destructive actions — Accepted
4. JSONL operation log by default — Accepted
5. Concurrent scanner with bounded workers — Accepted
6. Package manager abstraction w/ capability detection — Accepted
7. External commands must use timeout wrapper — Accepted
8. Built-in rules for MVP, extensibility in v1.x — Accepted

## 9. Open Questions + Default Decisions
1. Optimize per-DE vs minimal cross-distro
   Default: minimal cross-distro first.
2. Docker cleanup in purge default?
   Default: separate command in v1.x.
3. Clean to Trash vs permanent delete
   Default: permanent delete with strict confirmation (as PRD MVP).

## 10. Documentation Deliverables
1. `docs/IMPLEMENTATION_PLAN.md`
2. `docs/ARCHITECTURE.md`
3. `docs/SAFETY_MODEL.md`
4. `docs/TEST_STRATEGY.md`
5. `docs/RULESET_REFERENCE.md`
6. `docs/COMPATIBILITY_MATRIX.md`
7. `docs/RELEASE_PROCESS.md`
8. `docs/ROADMAP.md`

## 11. Success Criteria
- MVP commands stable and safe
- Zero blocked-path deletion incidents
- JSON output schema stable in CI
- Cross-distro smoke tests pass on Tier-1
- False-positive deletion near zero

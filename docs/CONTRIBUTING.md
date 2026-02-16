# Contributing

## Getting Started
1. Install Go (version to be specified when codebase is initialized).
2. Clone the repository.
3. Run tests locally before opening a PR.

## Development Workflow
- Keep changes scoped and focused.
- Follow safety-first design for any destructive action.
- Ensure `--dry-run` and `--json` outputs remain consistent.

## Testing Expectations
- Add unit tests for safety, rules, and scanner behavior.
- Update golden JSON tests if schema changes (with justification).
- Run integration tests with fixtures for filesystem behavior.

## Pull Request Checklist
- Tests pass locally.
- Docs updated when changing user-visible behavior.
- Operation log behavior remains intact.
- No telemetry added without explicit approval.

# Security Policy

Thank you for helping keep Talpa and its users safe.

## Supported Versions

Talpa is currently in active development. Security fixes are prioritized for:

- Latest `main` branch
- Latest tagged release

Older tags may not receive backported fixes.

## Reporting a Vulnerability

Please **do not open public issues** for suspected vulnerabilities.

Instead, report privately via:

- GitHub Security Advisories (preferred):
  - Repository → **Security** → **Report a vulnerability**

Include as much detail as possible:

- Affected version/commit
- Reproduction steps or proof-of-concept
- Impact assessment
- Any suggested remediation

## Response Expectations

We aim to:

- Acknowledge reports within **72 hours**
- Provide an initial triage status within **7 days**
- Release a fix according to severity and complexity

## Scope Notes

Given Talpa is a local system-maintenance CLI, important classes include:

- Unsafe path handling / traversal bypass
- Symlink-escape and TOCTOU-related issues
- Privilege boundary bypass
- Unsafe external command execution
- Data loss scenarios caused by destructive flows

## Disclosure Policy

We follow coordinated disclosure. Please allow time for triage and patching before public disclosure.

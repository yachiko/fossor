# Security Policy

## Supported Versions

Fossor is pre-1.0. Only the latest `0.x` release receives security fixes.

| Version | Supported |
| ------- | --------- |
| 0.1.x   | ✅        |
| < 0.1.0 | ❌        |

## Reporting a Vulnerability

Please report security vulnerabilities **privately** via GitHub Security Advisories:

1. Open the project's [Security tab](https://github.com/yachiko/fossor/security/advisories).
2. Click **Report a vulnerability**.
3. Provide a clear description, reproduction steps, and the impact you observed.

Do not open a public issue or PR for suspected vulnerabilities.

## What to Expect

- **Acknowledgement** within 5 business days.
- **Initial assessment** (severity, scope, affected versions) within 10 business days.
- **Fix or mitigation timeline** communicated once the assessment is complete.
- **Public disclosure** coordinated with the reporter; advisory and patched release published together.

## Threat Model

Fossor is a local TUI that shells out to the `git` binary on behalf of the user. The relevant threat surfaces are:

- **Repository discovery** scans a directory you point it at. Fossor does not follow symlinks outside the requested root.
- **Stale-lock cleanup** removes `.git/*.lock` files older than a threshold *and* not held by any live process (verified via `lsof` when available). It will not delete locks held by a running git process.
- **Interactive git commands** (commit editor, rebase -i, merge conflict resolution) suspend the TUI and hand the terminal to `git`. Fossor passes through your existing environment — `$EDITOR`, `$GIT_*`, credentials helpers, etc. are honored as configured.
- **`--open-cmd` / `$FOSSOR_OPEN_CMD`** runs the configured command with the selected repository path as its only argument. Set this to something you trust.

Fossor does not perform authentication. It assumes your git credentials are configured on the machine.

## Out of Scope

- Misconfiguration of `--open-cmd` to point at something destructive.
- Bugs in your installed `git` binary or credential helpers.
- Compromise of the host machine.

## Defensive Measures

- The binary is statically built and shipped with `-s -w` stripped symbols.
- All git invocations run with `exec.CommandContext` so they respect cancellation when the TUI shuts down.
- Bulk pull/fetch parallelism is capped at 8 to avoid resource exhaustion when scanning hundreds of repos.
- Stale-lock removal is gated behind both an mtime threshold and a `lsof` holder check; PRs that tighten this further are welcome.

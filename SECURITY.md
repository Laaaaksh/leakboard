# Security Policy

## Supported versions

Leakboard is a young project. Security fixes are made against the **latest
release** and `main` only.

| Version        | Supported |
| -------------- | --------- |
| latest release | yes       |
| older releases | no        |

## Reporting a vulnerability

Please do **not** open a public GitHub issue for anything you believe is a
security problem.

Use GitHub's private vulnerability reporting instead:

> https://github.com/Laaaaksh/leakboard/security/advisories/new

That link reaches the maintainer privately — the report, follow-up discussion,
and any fix coordination stay confidential until a patched release ships.

When reporting, please include:

- the Leakboard version or commit you're running
- how you deployed it (Docker Compose, bare binary, other)
- clear steps to reproduce

## What belongs in a report

Leakboard stores detected secrets (by design — you need the value to rotate
the credential) and holds a GitHub personal access token with read access to
every repo it scans. That makes its own trust boundary unusually important.
Things worth reporting:

- Any way for one authenticated dashboard user, or an unauthenticated
  request, to read another org's findings or a connection's access token.
- Session cookie handling: fixation, missing `HttpOnly`/`Secure`, or a way to
  forge a valid session token.
- A path where a repo's contents (a filename, a commit message, a secret
  value itself) causes command execution outside the intended `git`/`gitleaks`
  subprocess calls — Leakboard shells out to both.
- A webhook or connection URL that lets a user reach internal network
  services from the server (SSRF) via the outbound alert or GitHub API calls.
- The allowlist or "mark as false positive" feature suppressing a finding it
  shouldn't (a scoping bug that hides a real secret).
- Dependency vulnerabilities in the Go module graph or the frontend's npm
  packages with a real exploitation path in Leakboard's own usage.

Out of scope:

- Findings' secret values themselves being visible to a user who is already
  authenticated to the dashboard — that visibility is the product's job.
  Access control to the dashboard, not the data within it once authenticated,
  is the boundary that matters here.
- Reports that require an attacker to already have write access to a
  connected repo (they can already introduce whatever they like into its
  history).
- Vulnerabilities in gitleaks itself — please report those to
  [gitleaks/gitleaks](https://github.com/gitleaks/gitleaks/security).

## Credits

Reporters who wish to be credited in a fix's release notes may say so in the
private report; otherwise reports are handled without attribution.

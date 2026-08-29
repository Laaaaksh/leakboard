## What does this PR do?

<!-- One or two sentences describing the change. -->

## Checklist

- [ ] `make lint` and `make test` pass locally
- [ ] Tests added or updated for the changed behavior
- [ ] User-facing docs updated if behavior changed (README.md, CHANGELOG.md)
- [ ] If this touches scanning/dedup/allowlist logic, a test proves the new
      behavior against a real gitleaks run (see `internal/scanner/incremental_test.go`
      for the pattern) — not just a mocked assertion

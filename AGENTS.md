# Project agent memory

This file is the project's committed home for project-intrinsic agent knowledge: build, test, release, architecture, and sharp-edge notes that should travel with the code.

## Architecture

- Single Go binary + Postgres. The scan pipeline is split: `internal/gitutil` manages bare
  mirror clones and computes incremental git-log ranges; `internal/gitleaks` shells out to the
  real `gitleaks` CLI and parses its JSON report; `internal/scanner` orchestrates the two,
  applies the allowlist, persists findings, and dispatches alerts. Detection rules live in
  gitleaks upstream — this repo never reimplements them.
- Dedup key is gitleaks' own `Fingerprint` (`commit:file:rule:line`), enforced by a unique
  constraint on `(repo_id, fingerprint)` and an `ON CONFLICT ... RETURNING (xmax = 0)` upsert
  (`internal/store/findings.go`) to distinguish a genuinely new finding from a rescan hit
  without a second query.
- Incremental scanning diffs the *current* ref tips against the ref tips stored at the last
  successful scan (`gitutil.BuildIncrementalLogOpts`), not just the default branch — a secret
  on any branch is caught on the next scan, not only ones merged to `master`.
- The React frontend is embedded via `internal/webui` (`go:embed`), and the **real built
  output is committed** to `internal/webui/dist` — not a placeholder — so `go build`/`go
  install` produce a working dashboard with no Node install required. Any PR touching `web/`
  must run `make build-frontend` and commit the resulting `internal/webui/dist` diff; CI's
  "Frontend" job rebuilds and diffs it, failing if it's stale.

## Sharp edges

- The SPA fallback (`internal/api/spa.go`) must serve `index.html`'s bytes directly, not
  delegate to `http.FileServer` with the request path rewritten to `/index.html` — FileServer
  treats any path ending in `index.html` as needing a redirect to its parent directory, which
  silently bounces every client-side route (`/repos`, `/findings/42`, ...) back to `/`.
  `internal/api/spa_test.go` guards this.
- Postgres-backed tests (`internal/store`, `internal/api`) truncate shared tables and must run
  with `go test -p 1` (see `Makefile`) or they race each other across packages. They're skipped
  automatically, not failed, when `TEST_DATABASE_URL` is unset.
- `.golangci.yml`'s `errcheck.exclude-functions` list covers `Close`/`Rollback` on stdlib types;
  a custom type's `Close` (e.g. `*store.Store`) isn't covered by that and needs an explicit
  `_ = x.Close()` or `defer func() { _ = x.Close() }()`.
- The Docker image's named volume for repo mirrors must be `chown`'d to the `leakboard` user
  *before* `USER leakboard` in the Dockerfile — Docker only inherits a fresh named volume's
  initial ownership from the image path it overlays, so skipping this makes every scan fail
  with a permission error against a completely empty-looking volume.

## Demo recording

- `make demo` (`scripts/record-demo/`) boots a fresh stack, seeds a real git fixture repo,
  drives the actual UI with Playwright, and produces `docs/assets/demo.mp4`/`demo.gif`. It's a
  dev-only Node package (own `package.json`) never imported by the product build. See
  `scripts/record-demo/README.md`.
- The seed repo is made reachable to the container via `docker compose cp` into
  `/home/leakboard/data/seed-repo` + a root `chown` to the `leakboard` user, then tracked
  through the real "Add a single repo" UI as `file:///home/leakboard/data/seed-repo` —
  Leakboard's `gitutil` just shells out to `git clone --mirror`, so any reachable clone URL
  works and no GitHub token or network access is needed for the demo.
- Gitleaks' default config allowlists the literal AWS docs placeholder credential (anything
  matching `.+EXAMPLE$`), so a fixture using that exact well-known value won't be detected —
  `fixture.mjs` uses a different fake key that still matches the `AKIA` + 16-char
  `[A-Z2-7]` rule shape.

## Maintaining this file

Keep this file for knowledge useful to almost every future agent session in this project.
Do not repeat what the codebase already shows; point to the authoritative file or command instead.
Prefer rewriting or pruning existing entries over appending new ones.
When updating this file, preserve this bar for all agents and keep entries concise.

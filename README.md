<div align="center">

<img src="docs/assets/leakboard-banner.svg" alt="Leakboard" width="640">

**Leakboard** — the tracking layer GitGuardian sells, built on the free scanners everyone
already trusts. A self-hosted dashboard that runs [Gitleaks](https://github.com/gitleaks/gitleaks)
on a schedule across every repo in your GitHub org, deduplicates findings so you're never
re-alerted on the same leaked secret twice, and pages Slack, Discord, or a webhook only when
something genuinely new shows up.

[![Star this repo](https://img.shields.io/github/stars/Laaaaksh/leakboard?style=for-the-badge&logo=github&label=star%20this%20repo&color=yellow)](https://github.com/Laaaaksh/leakboard/stargazers)
[![Built on Gitleaks](https://img.shields.io/badge/built_on-Gitleaks-00ADD8?style=for-the-badge&logo=git&logoColor=white)](https://github.com/gitleaks/gitleaks)

[![CI](https://github.com/Laaaaksh/leakboard/actions/workflows/ci.yml/badge.svg)](https://github.com/Laaaaksh/leakboard/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Laaaaksh/leakboard?color=green&display_name=tag)](https://github.com/Laaaaksh/leakboard/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-purple.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26+-00ADD8?logo=go&logoColor=white)](go.mod)
[![Docker](https://img.shields.io/badge/docker-compose-2496ED?logo=docker&logoColor=white)](docker-compose.yml)

**[Install](#install) • [Usage](#usage) • [Configuration](#configuration) • [Changelog](CHANGELOG.md) • [Contributing](CONTRIBUTING.md) • [License](#license)**

**[Code of conduct](CODE_OF_CONDUCT.md) • [Security](SECURITY.md)**

</div>

## What it does

- Scans every repo in a connected GitHub org on a schedule, using Gitleaks' full rule set —
  no reimplemented detection logic, no new false-positive surface to tune from scratch.
- After the first full-history baseline scan, incrementally scans only the commits introduced
  since the last run, across every branch — not just the default one.
- Deduplicates findings by commit + file + rule, so a secret still sitting in old history
  never re-triggers an alert on every scheduled run — only genuinely new findings do.
- Tracks each finding's status — new, acknowledged, resolved, or false positive — with the
  file, line, commit, and author it was introduced by.
- Fires a Slack, Discord, or generic JSON webhook exactly once per new finding.
- Shares an org-wide allowlist (by rule ID, path glob, or regex) so a known-safe pattern is
  suppressed everywhere, not re-triaged per repo.
- Ships as one Go binary with the dashboard frontend embedded, backed by Postgres — no
  scanning-as-a-service, no data leaving your infrastructure.

<p align="center">
  <img src="docs/assets/leakboard-demo.gif" alt="Leakboard: a new secret is committed to a tracked repo, a scan is triggered from the dashboard, and the finding count updates live with the new leak at the top" width="860">
</p>

*Real capture: a token is committed to a tracked repo, "Scan now" is clicked from the
dashboard, and the new finding appears at the top of the findings table seconds later —
while the pre-existing findings underneath do not re-trigger anything.*

## Requirements

- A GitHub personal access token (classic or fine-grained) with read access to the repos you
  want scanned — used to list an org's repos and clone them. Leakboard does not use a
  registered GitHub App, so there's no webhook endpoint or app manifest to host.
- Postgres 14+ (bundled in `docker-compose.yml`).
- The [gitleaks](https://github.com/gitleaks/gitleaks) binary on `PATH` if you're running the
  Go binary directly instead of Docker (the Docker image already includes it).
- Enough disk for a bare mirror clone of every repo you track — Leakboard keeps a local
  git mirror per repo to scan incrementally, at roughly the size of that repo's own `.git`.
- Node 20+ — **only** if you build from source with `make build`, which builds the dashboard
  frontend before the Go binary. Skip it entirely by running `go build ./cmd/leakboard`
  instead: `internal/webui/dist` ships a real, committed frontend build, so the plain Go build
  produces a working dashboard with no Node install at all. Docker Compose needs no Node
  either way — the multi-stage `Dockerfile` builds the frontend for you.

## Install

The fastest path is Docker Compose, which also runs Postgres for you:

```bash
git clone https://github.com/Laaaaksh/leakboard.git
cd leakboard
export LEAKBOARD_SESSION_SECRET="$(openssl rand -hex 32)"
docker compose up -d
```

The first `up -d` builds the image from scratch (Go build, gitleaks fetch, Debian package
installs) and can take a few minutes with no progress output — it hasn't hung. Leakboard is
then on `http://localhost:8080`; the first visit creates the single admin account for this
instance. Something else already listening on 8080? Set `LEAKBOARD_PORT` before bringing it up
(e.g. `export LEAKBOARD_PORT=8081`) to change the host-side port without editing
`docker-compose.yml`.

To build and run from source instead (no Docker, no Node — see Requirements above):

```bash
go mod download
go build -o leakboard ./cmd/leakboard
export LEAKBOARD_DATABASE_URL="postgres://user:pass@localhost:5432/leakboard?sslmode=disable"
export LEAKBOARD_SESSION_SECRET="$(openssl rand -hex 32)"
./leakboard
```

## Usage

1. Open the dashboard and create the admin account (first visit only).
2. Go to **Repos** → **Connect a GitHub organization**, and give it a display name, the org's
   login, and a personal access token with read access to its repos. Every non-archived repo
   the token can see is added and scanned on the configured interval (default: every 5
   minutes). Prefer a single repo instead? Use **Add a single repo** with any git clone URL.
3. Findings show up on the **Findings** page as scans complete — filter by status, open one for
   the file/line/commit and a masked secret preview (reveal it explicitly when you need the
   real value to rotate the credential).
4. Acknowledge, resolve, or mark a finding a false positive. False-positive additionally
   allowlists that exact finding's fingerprint, so it won't resurface.
5. Add a Slack, Discord, or generic webhook under **Settings** to get paged the moment a new
   secret is introduced.

<p align="center">
  <img src="docs/assets/dashboard-screenshot.png" alt="Leakboard Findings page showing findings by status with rule, repo, location, and first-seen columns" width="420">
  <img src="docs/assets/finding-detail-screenshot.png" alt="Leakboard finding detail showing file, line, commit, and a masked secret preview" width="420">
</p>

## Configuration

Leakboard is configured entirely by environment variables — see `docker-compose.yml` for the
defaults Docker Compose sets.

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `LEAKBOARD_DATABASE_URL` | yes | — | Postgres connection string |
| `LEAKBOARD_SESSION_SECRET` | yes | — | Any random string; signs nothing sensitive today but is validated at startup so it's never forgotten before exposing the instance |
| `LEAKBOARD_ADDR` | no | `:8080` | HTTP listen address |
| `LEAKBOARD_BASE_URL` | no | `http://localhost:8080` | Used to build links in webhook alerts |
| `LEAKBOARD_WORKDIR` | no | `./data/repos` | Where repo mirror clones are stored |
| `LEAKBOARD_GITLEAKS_PATH` | no | `gitleaks` | Path to the gitleaks binary |
| `LEAKBOARD_SCAN_INTERVAL_SECONDS` | no | `300` | How often the scheduler checks for repos due a scan |

`LEAKBOARD_PORT` is a separate, Compose-only variable (not read by the binary itself) that
picks the **host-side** port `docker-compose.yml` publishes; the app inside the container
always listens on `LEAKBOARD_ADDR`.

## Limits

Leakboard is young and scoped narrow on purpose. Plainly, as of today:

- **One admin account per instance.** There's no multi-user login or role-based access —
  whoever completes first-run setup can see every finding and every revealed secret value.
- **GitHub only.** No GitLab or Bitbucket support; both the org-connect flow and "Add a single
  repo" expect a GitHub-hosted remote.
- **Detects and alerts, doesn't remediate.** Leakboard finds a leaked secret and tells you; it
  does not rotate or revoke credentials with AWS, Stripe, or any other provider.
- **Allowlist and rule changes aren't retroactive.** Incremental scans compare every ref
  currently in a repo's mirror against the ref tips seen at the last successful scan, so new
  commits on any branch — not just the default one — are caught. What they do *not* do is
  re-check history already scanned once for a Gitleaks rule added after the fact, or a new
  org-wide allowlist rule you add later — an allowlist rule only suppresses future findings, it
  never removes one already in the dashboard. A manual "Scan now" only scans commits since the
  last successful scan, never full history again; delete and re-add a repo for a fresh
  full-history baseline.

## Changelog

Notable changes per release live in [CHANGELOG.md](CHANGELOG.md).

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md).

Note: a bare `go test ./...` silently skips the Postgres-backed suites in `internal/store` and
`internal/api` — the ones covering dedup, auth, and the allowlist — unless `TEST_DATABASE_URL`
is set. See [CONTRIBUTING.md](CONTRIBUTING.md#contribution-workflow) for how to set it.

## Security

Found a security issue? Please report it privately — see [SECURITY.md](SECURITY.md).

## Star this repo

If Leakboard saves you from paying per-seat for something two free CLIs already do most of,
[leave a star](https://github.com/Laaaaksh/leakboard/stargazers) — it helps other people find it.

## License

MIT — see [LICENSE](LICENSE).

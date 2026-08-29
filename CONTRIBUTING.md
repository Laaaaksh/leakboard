# Contributing to Leakboard

Thank you for your interest in contributing. Leakboard is a self-hosted
secret-scanning dashboard, open source under the MIT license.

## Getting started

```bash
git clone https://github.com/<your-username>/leakboard.git   # your fork, see below
cd leakboard
go mod download
docker run -d --name leakboard-dev -e POSTGRES_PASSWORD=leakboard \
  -e POSTGRES_DB=leakboard -p 5432:5432 postgres:16-alpine
export LEAKBOARD_DATABASE_URL="postgres://postgres:leakboard@localhost:5432/leakboard?sslmode=disable"
export LEAKBOARD_SESSION_SECRET="$(openssl rand -hex 32)"
make build
make test
```

## Requirements

- Go 1.26+
- Node 20+ (only needed to build the dashboard frontend — `go test` alone
  does not need it)
- [gitleaks](https://github.com/gitleaks/gitleaks) on your `PATH`
  (`brew install gitleaks`, or see its releases page) — the scanner and its
  tests shell out to the real binary
- Docker, for a local Postgres instance and for `TEST_DATABASE_URL`-gated tests

## Contribution workflow

The `master` branch is protected: every change lands through a pull request, required
status checks must pass, and protection is enforced for everyone — including the
maintainer. There are no direct pushes to `master`.

1. Fork the repo on GitHub, then clone your fork (command above).
2. Create a descriptively named feature branch from `master`.
3. Make your changes as small, focused commits, each leaving the tree buildable.
4. Run `make lint` and `make test` — both must pass. Postgres-backed tests in
   `internal/store` and `internal/api` are skipped automatically if
   `TEST_DATABASE_URL` isn't set; set it to actually exercise them:
   ```bash
   export TEST_DATABASE_URL="postgres://postgres:leakboard@localhost:5432/leakboard_test?sslmode=disable"
   ```
5. If your change is user-facing (a feature, fix, or behavior change), add one
   bullet under the `Unreleased` heading in [CHANGELOG.md](CHANGELOG.md).
6. If you touched anything under `web/`, run `make build-frontend` and commit the
   resulting change to `internal/webui/dist` — it's a real, committed build (not a
   placeholder) so `go build` alone produces a working dashboard, and CI fails if it's stale.
7. Push the branch to your fork.
8. Open a pull request against `master` here.

A PR can merge only when every required check passes (`Test`, `Lint`) and all
conversation threads are resolved.

## Releases

Releases are cut by pushing a tag; GitHub Actions does the rest
(`.github/workflows/release.yml`):

1. Make sure every user-facing change since the last release has a bullet under
   `Unreleased` in [CHANGELOG.md](CHANGELOG.md) (step 5 of the workflow above).
2. Give the release its own changelog section: insert `## [x.y.z] - YYYY-MM-DD`
   above the (now empty) `## [Unreleased]` heading, following the format of the
   existing sections, and update the compare links at the bottom of the file —
   add `[x.y.z]: https://github.com/Laaaaksh/leakboard/compare/v<prev>...vx.y.z`
   and repoint `[Unreleased]` at `compare/vx.y.z...HEAD`.
3. Land those changelog edits on `master` through a pull request (see the
   contribution workflow above), then tag and push:

   ```bash
   git tag vx.y.z && git push origin vx.y.z
   ```

The release workflow extracts the tagged version's CHANGELOG section as the
GitHub release notes (failing loudly rather than publishing empty notes if
that section is missing), then builds and pushes a multi-arch Docker image to
`ghcr.io/laaaaksh/leakboard` tagged both `x.y.z` and `latest`.

## Code style

- Standard `gofmt` formatting on the Go side (enforced by CI); the frontend
  follows `oxlint`'s defaults (also enforced by CI).
- Follow the existing package structure: HTTP handlers in `internal/api`,
  persistence in `internal/store`, the scan pipeline split across
  `internal/scanner` (orchestration), `internal/gitutil` (mirror clones and
  incremental ranges), and `internal/gitleaks` (the CLI wrapper).
- Leakboard shells out to `git` and `gitleaks` rather than reimplementing
  either — keep it that way. Detection-rule changes belong upstream in
  [gitleaks](https://github.com/gitleaks/gitleaks), not here.
- A Postgres-backed test belongs next to the code it tests, gated on
  `TEST_DATABASE_URL` being set (see `internal/store/store_test.go` for the
  pattern) — never skip writing the test just because CI needs a service
  container for it.

## Reporting issues

Please open a GitHub issue before starting large changes or proposing new
features, so scope and approach can be settled before code is written. Bug
reports should include:
- Leakboard version or commit
- How you deployed it (Docker Compose, bare binary)
- Steps to reproduce
- What you expected vs what happened

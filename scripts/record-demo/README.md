# record-demo

Re-records `docs/assets/demo.mp4` / `docs/assets/demo.gif` against a real, freshly-booted
Leakboard instance. Dev-only tooling - not part of the product build, and not imported by
anything under `internal/` or `web/`.

## Run it

```bash
make demo
```

That's `scripts/record-demo/run.sh`, which:

1. `docker compose down -v && docker compose up -d --build` - a clean instance, so the
   recording always starts from the real first-run "create admin account" screen.
2. Waits for `/api/session` to respond.
3. Seeds a small local git repo with fake, non-functional credentials (an AWS-shaped key, a
   Stripe-test-shaped key) and copies it into the running container's data volume, so it's
   reachable at a `file://` path - a real git remote Leakboard clones the normal way, not a
   database fixture.
4. Drives the actual UI with Playwright (`record.mjs`): create account, add the repo, scan it,
   triage one finding as a true positive and one as a false positive, add an allowlist rule,
   and record video throughout.
5. Converts the capture with ffmpeg into `docs/assets/demo.mp4` (H.264) and
   `docs/assets/demo.gif` (960px, ~12fps, under 10MB - drops fps automatically if needed).

Override the port if 8080 is taken locally: `LEAKBOARD_PORT=8081 make demo`.

## Run steps individually

Useful when iterating on one stage without re-booting the whole stack:

```bash
cd scripts/record-demo
npm install && npx playwright install chromium   # first time only
npm run seed      # (re)plant the fixture repo in the running container
npm run record    # drive the UI, produce output/demo.webm
npm run convert   # encode output/demo.webm into docs/assets/
```

`seed.mjs` and `record.mjs` assume a Leakboard instance is already reachable at
`DEMO_BASE_URL` (default `http://localhost:8080`) and freshly reset - `record.mjs` always goes
through first-run setup, so a second run against a non-reset instance will fail at that step.

## Files

- `fixture.mjs` - builds the local git repo with planted fake secrets.
- `seed.mjs` - copies that repo into the running container (`docker compose cp` + `chown`).
- `record.mjs` - the Playwright walkthrough; produces `output/demo.webm`.
- `convert.sh` - ffmpeg conversion into `docs/assets/demo.mp4` and `demo.gif`.
- `run.sh` - orchestrates all of the above end to end.

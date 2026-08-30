#!/usr/bin/env bash
# Boots a fresh Leakboard stack, seeds it, records the demo walkthrough, and
# converts the capture into docs/assets/. One command, safe to re-run.
#
# Env overrides:
#   LEAKBOARD_PORT           host port to publish (default: 8080)
#   LEAKBOARD_SESSION_SECRET session secret (default: random each run)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PORT="${LEAKBOARD_PORT:-8080}"

cd "$REPO_ROOT"
export LEAKBOARD_PORT="$PORT"
export LEAKBOARD_SESSION_SECRET="${LEAKBOARD_SESSION_SECRET:-$(openssl rand -hex 32)}"

echo "==> Resetting the stack for a deterministic first-run recording"
docker compose down -v --remove-orphans

echo "==> Building and starting Leakboard on http://localhost:${PORT}"
docker compose up -d --build

echo "==> Waiting for Leakboard to become reachable"
for _ in $(seq 1 60); do
  if curl -sf "http://localhost:${PORT}/api/session" >/dev/null; then
    break
  fi
  sleep 2
done
curl -sf "http://localhost:${PORT}/api/session" >/dev/null || {
  echo "error: Leakboard never became reachable on port ${PORT}" >&2
  exit 1
}

cd "$SCRIPT_DIR"
if [[ ! -d node_modules ]]; then
  echo "==> Installing recorder dependencies"
  npm install
  npx playwright install --with-deps chromium
fi

export DEMO_BASE_URL="http://localhost:${PORT}"
echo "==> Seeding the demo repo into the container"
npm run seed
echo "==> Recording the walkthrough"
npm run record
echo "==> Converting to docs/assets/demo.mp4 and demo.gif"
npm run convert

echo "==> Done. docs/assets/demo.mp4 and docs/assets/demo.gif are up to date."

#!/usr/bin/env bash
# Converts the raw Playwright capture (output/demo.webm) into the two
# committed assets and drops them into docs/assets/, replacing whatever was
# there before. Requires ffmpeg/ffprobe on PATH.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SRC="$SCRIPT_DIR/output/demo.webm"
ASSETS_DIR="$REPO_ROOT/docs/assets"
GIF_FPS="${DEMO_GIF_FPS:-12}"
GIF_MAX_MB=10

if [[ ! -f "$SRC" ]]; then
  echo "error: $SRC not found - run \`npm run record\` first" >&2
  exit 1
fi

mkdir -p "$ASSETS_DIR"

echo "Encoding docs/assets/demo.mp4 ..."
ffmpeg -y -i "$SRC" \
  -vf "scale=1280:-2" \
  -c:v libx264 -pix_fmt yuv420p -movflags +faststart \
  "$ASSETS_DIR/demo.mp4"

PALETTE="$SCRIPT_DIR/output/palette.png"

encode_gif() {
  local fps="$1"
  ffmpeg -y -i "$SRC" -vf "fps=${fps},scale=960:-2:flags=lanczos,palettegen" "$PALETTE"
  ffmpeg -y -i "$SRC" -i "$PALETTE" \
    -lavfi "fps=${fps},scale=960:-2:flags=lanczos[x];[x][1:v]paletteuse" \
    "$ASSETS_DIR/demo.gif"
}

echo "Encoding docs/assets/demo.gif at ${GIF_FPS}fps ..."
encode_gif "$GIF_FPS"

size_mb() { du -m "$1" | cut -f1; }

while [[ "$(size_mb "$ASSETS_DIR/demo.gif")" -gt "$GIF_MAX_MB" && "$GIF_FPS" -gt 6 ]]; do
  GIF_FPS=$((GIF_FPS - 2))
  echo "GIF over ${GIF_MAX_MB}MB, retrying at ${GIF_FPS}fps ..."
  encode_gif "$GIF_FPS"
done

FINAL_MB="$(size_mb "$ASSETS_DIR/demo.gif")"
if [[ "$FINAL_MB" -gt "$GIF_MAX_MB" ]]; then
  echo "error: demo.gif is still ${FINAL_MB}MB after dropping to ${GIF_FPS}fps - shorten the walkthrough in record.mjs" >&2
  exit 1
fi

rm -f "$PALETTE"
echo "Done: docs/assets/demo.mp4, docs/assets/demo.gif (${FINAL_MB}MB)"

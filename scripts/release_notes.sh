#!/usr/bin/env bash
# Extracts one version's section from CHANGELOG.md and writes it to the given
# output path, for use as a GitHub release's body. Fails rather than
# publishing empty notes if the version has no changelog section yet.
set -euo pipefail

out_path="${1:?usage: release_notes.sh <output-path> <version>}"
version="${2:?usage: release_notes.sh <output-path> <version>}"

awk -v version="$version" '
  $0 ~ "^## \\[" version "\\]" { found=1; next }
  found && /^## \[/ { exit }
  found { print }
' CHANGELOG.md > "$out_path"

if [ ! -s "$out_path" ]; then
  echo "error: no CHANGELOG.md section found for version $version (expected a '## [$version]' heading)" >&2
  exit 1
fi

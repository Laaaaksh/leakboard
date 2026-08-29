// Package webui embeds the built React dashboard so the whole product ships
// as one Go binary. dist/ holds a committed, real build of web/ (not a
// placeholder) so `go build`/`go install` work with no Node install; run
// `make build-frontend` and commit the result after any change under web/ —
// CI rebuilds and diffs dist/ to catch a stale commit.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// FS returns the embedded frontend build, rooted at dist/ so paths match
// what the HTTP server expects ("/" -> dist/index.html).
func FS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

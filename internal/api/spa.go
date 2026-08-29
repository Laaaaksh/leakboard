package api

import (
	"io"
	"io/fs"
	"net/http"
)

// spaHandler serves static files from fsys, falling back to index.html's
// contents for any path that doesn't match a real file, so client-side
// routes (e.g. /repos, /findings/42) resolve to the app shell instead of a
// 404. The fallback reads index.html directly rather than delegating to
// http.FileServer with a rewritten path: FileServer treats any request path
// ending in "index.html" as needing a redirect to its parent directory,
// which would bounce every client-side route straight back to "/".
func spaHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		if f, err := fsys.Open(path[1:]); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		serveIndex(w, fsys)
	})
}

func serveIndex(w http.ResponseWriter, fsys fs.FS) {
	f, err := fsys.Open("index.html")
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	defer func() { _ = f.Close() }()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, f)
}

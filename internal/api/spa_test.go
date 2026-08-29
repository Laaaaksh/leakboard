package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// TestSpaHandler_ClientRouteServesIndexWithoutRedirect guards against a real
// bug: rewriting the fallback request's path to "/index.html" and handing
// it to http.FileServer trips Go's own "requests ending in index.html
// redirect to their parent directory" behavior, bouncing every client-side
// route (e.g. /repos) back to "/" instead of serving the app shell.
func TestSpaHandler_ClientRouteServesIndexWithoutRedirect(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<html>app shell</html>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('real asset')")},
	}
	handler := spaHandler(fsys)

	for _, path := range []string{"/", "/repos", "/findings/42", "/allowlist"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("path %s: got status %d, want 200", path, rec.Code)
		}
		if got := rec.Body.String(); got != "<html>app shell</html>" {
			t.Errorf("path %s: got body %q, want the app shell", path, got)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("real asset: got status %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "console.log('real asset')" {
		t.Errorf("real asset: got body %q, want the real file contents", got)
	}
}

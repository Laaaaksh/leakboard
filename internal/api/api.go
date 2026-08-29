// Package api implements Leakboard's JSON HTTP API and serves the built
// React dashboard as static files with an SPA fallback.
package api

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/Laaaaksh/leakboard/internal/alert"
	"github.com/Laaaaksh/leakboard/internal/auth"
	"github.com/Laaaaksh/leakboard/internal/gitleaks"
	"github.com/Laaaaksh/leakboard/internal/scanner"
	"github.com/Laaaaksh/leakboard/internal/store"
)

// Server holds the dependencies every HTTP handler needs.
type Server struct {
	Store    *store.Store
	Scanner  *scanner.Scanner
	Gitleaks *gitleaks.Scanner
	Alerts   *alert.Dispatcher
	Log      *slog.Logger
	WorkDir  string
	Secure   bool // whether to mark session cookies Secure (true behind HTTPS)
}

// Routes builds the full handler: the JSON API under /api/, and the SPA
// (frontendFS, which must contain an index.html at its root) for everything
// else, so client-side routes like /findings/42 still load the app shell.
func (s *Server) Routes(frontendFS fs.FS) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/session", s.handleSession)
	mux.HandleFunc("POST /api/setup", s.handleSetup)
	mux.HandleFunc("POST /api/login", s.handleLogin)
	mux.HandleFunc("POST /api/logout", s.handleLogout)

	mux.HandleFunc("GET /api/stats", s.withAuth(s.handleStats))
	mux.HandleFunc("GET /api/scans", s.withAuth(s.handleRecentScans))

	mux.HandleFunc("GET /api/findings", s.withAuth(s.handleListFindings))
	mux.HandleFunc("GET /api/findings/{id}", s.withAuth(s.handleGetFinding))
	mux.HandleFunc("POST /api/findings/{id}/status", s.withAuth(s.handleUpdateFindingStatus))

	mux.HandleFunc("GET /api/repos", s.withAuth(s.handleListRepos))
	mux.HandleFunc("POST /api/repos", s.withAuth(s.handleCreateRepo))
	mux.HandleFunc("DELETE /api/repos/{id}", s.withAuth(s.handleDeleteRepo))
	mux.HandleFunc("POST /api/repos/{id}/scan", s.withAuth(s.handleScanRepoNow))

	mux.HandleFunc("GET /api/connections", s.withAuth(s.handleListConnections))
	mux.HandleFunc("POST /api/connections", s.withAuth(s.handleCreateConnection))
	mux.HandleFunc("DELETE /api/connections/{id}", s.withAuth(s.handleDeleteConnection))
	mux.HandleFunc("POST /api/connections/{id}/sync", s.withAuth(s.handleSyncConnection))

	mux.HandleFunc("GET /api/allowlist", s.withAuth(s.handleListAllowlist))
	mux.HandleFunc("POST /api/allowlist", s.withAuth(s.handleCreateAllowlist))
	mux.HandleFunc("DELETE /api/allowlist/{id}", s.withAuth(s.handleDeleteAllowlist))

	mux.HandleFunc("GET /api/webhooks", s.withAuth(s.handleListWebhooks))
	mux.HandleFunc("POST /api/webhooks", s.withAuth(s.handleCreateWebhook))
	mux.HandleFunc("DELETE /api/webhooks/{id}", s.withAuth(s.handleDeleteWebhook))

	mux.Handle("/", spaHandler(frontendFS))

	return auth.Middleware(s.Store)(mux)
}

func (s *Server) withAuth(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := auth.RequireUser(w, r); !ok {
			return
		}
		h(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

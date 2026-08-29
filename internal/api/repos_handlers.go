package api

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Laaaaksh/leakboard/internal/store"
)

type repoJSON struct {
	ID               int64   `json:"id"`
	Name             string  `json:"name"`
	CloneURL         string  `json:"cloneUrl"`
	DefaultBranch    string  `json:"defaultBranch"`
	LastScannedAt    *string `json:"lastScannedAt"`
	LastScanError    string  `json:"lastScanError"`
	ScanIntervalSecs int     `json:"scanIntervalSecs"`
}

func toRepoJSON(r store.Repo) repoJSON {
	out := repoJSON{
		ID: r.ID, Name: r.Name, CloneURL: redactCloneURL(r.CloneURL), DefaultBranch: r.DefaultBranch,
		LastScanError: r.LastScanError, ScanIntervalSecs: r.ScanIntervalSecs,
	}
	if r.LastScannedAt != nil {
		s := r.LastScannedAt.Format(timeFormat)
		out.LastScannedAt = &s
	}
	return out
}

// redactCloneURL strips any embedded x-access-token credential before a
// clone URL ever reaches an API response.
func redactCloneURL(url string) string {
	if i := strings.Index(url, "@"); i != -1 && strings.Contains(url[:i], "x-access-token:") {
		return "https://" + url[i+1:]
	}
	return url
}

func (s *Server) handleListRepos(w http.ResponseWriter, r *http.Request) {
	repos, err := s.Store.ListRepos(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "listing repos")
		return
	}
	out := make([]repoJSON, len(repos))
	for i, repo := range repos {
		out[i] = toRepoJSON(repo)
	}
	writeJSON(w, http.StatusOK, out)
}

type createRepoRequest struct {
	Name          string `json:"name"`
	CloneURL      string `json:"cloneUrl"`
	DefaultBranch string `json:"defaultBranch"`
}

// handleCreateRepo adds a single repo by its clone URL, for anyone who
// wants to track a repo outside a connected GitHub org (a personal repo, a
// non-GitHub remote, or a quick trial without setting up a connection).
func (s *Server) handleCreateRepo(w http.ResponseWriter, r *http.Request) {
	var req createRepoRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.CloneURL = strings.TrimSpace(req.CloneURL)
	if req.Name == "" || req.CloneURL == "" {
		writeError(w, http.StatusBadRequest, "name and cloneUrl are required")
		return
	}
	if req.DefaultBranch == "" {
		req.DefaultBranch = "main"
	}

	repo, err := s.Store.CreateRepo(r.Context(), store.NewRepo{
		Name: req.Name, CloneURL: req.CloneURL, DefaultBranch: req.DefaultBranch,
		MirrorPath: mirrorPathFor(s.WorkDir, req.CloneURL),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "creating repo")
		return
	}
	writeJSON(w, http.StatusCreated, toRepoJSON(repo))
}

// mirrorPathFor derives a stable, filesystem-safe local path for a repo's
// mirror clone from its clone URL, so the same repo always lands in the
// same place across restarts.
func mirrorPathFor(workDir, cloneURL string) string {
	sum := sha256.Sum256([]byte(cloneURL))
	return filepath.Join(workDir, hex.EncodeToString(sum[:])+".git")
}

func (s *Server) handleDeleteRepo(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.Store.DeleteRepo(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "deleting repo")
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

// handleScanRepoNow triggers an immediate scan and waits for it to finish,
// so the UI's "Scan now" button can show a real result rather than polling.
func (s *Server) handleScanRepoNow(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	repo, err := s.Store.RepoByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "repo not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "loading repo")
		return
	}

	newCount, err := s.Scanner.ScanRepo(r.Context(), repo)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "scan failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"newFindings": newCount})
}

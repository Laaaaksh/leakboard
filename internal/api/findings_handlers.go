package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Laaaaksh/leakboard/internal/store"
)

// findingJSON masks the stored secret to a short preview by default; the
// full value is only included when the client explicitly asks with
// ?reveal=1, so a shoulder-surf of the dashboard doesn't leak every secret.
type findingJSON struct {
	ID           int64  `json:"id"`
	RepoID       int64  `json:"repoId"`
	RepoName     string `json:"repoName"`
	Fingerprint  string `json:"fingerprint"`
	RuleID       string `json:"ruleId"`
	Description  string `json:"description"`
	FilePath     string `json:"filePath"`
	StartLine    int    `json:"startLine"`
	EndLine      int    `json:"endLine"`
	CommitSHA    string `json:"commitSha"`
	CommitAuthor string `json:"commitAuthor"`
	CommitEmail  string `json:"commitEmail"`
	Secret       string `json:"secret"`
	Status       string `json:"status"`
	FirstSeenAt  string `json:"firstSeenAt"`
	LastSeenAt   string `json:"lastSeenAt"`
}

func toFindingJSON(f store.Finding, reveal bool) findingJSON {
	return findingJSON{
		ID: f.ID, RepoID: f.RepoID, RepoName: f.RepoName, Fingerprint: f.Fingerprint,
		RuleID: f.RuleID, Description: f.Description, FilePath: f.FilePath,
		StartLine: f.StartLine, EndLine: f.EndLine, CommitSHA: f.CommitSHA,
		CommitAuthor: f.CommitAuthor, CommitEmail: f.CommitEmail,
		Secret: maskSecret(f.Secret, reveal), Status: string(f.Status),
		FirstSeenAt: f.FirstSeenAt.Format(timeFormat), LastSeenAt: f.LastSeenAt.Format(timeFormat),
	}
}

const timeFormat = "2006-01-02T15:04:05Z07:00"

func maskSecret(secret string, reveal bool) string {
	if reveal {
		return secret
	}
	if len(secret) <= 8 {
		return "••••••••"
	}
	return secret[:4] + "…" + secret[len(secret)-4:]
}

func (s *Server) handleListFindings(w http.ResponseWriter, r *http.Request) {
	filter := store.FindingFilter{
		Status: store.FindingStatus(r.URL.Query().Get("status")),
		RuleID: r.URL.Query().Get("rule_id"),
	}
	if repoID := r.URL.Query().Get("repo_id"); repoID != "" {
		id, err := strconv.ParseInt(repoID, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid repo_id")
			return
		}
		filter.RepoID = &id
	}

	findings, err := s.Store.ListFindings(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "listing findings")
		return
	}

	out := make([]findingJSON, len(findings))
	for i, f := range findings {
		out[i] = toFindingJSON(f, false)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetFinding(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	f, err := s.Store.FindingByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "finding not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "loading finding")
		return
	}
	reveal := r.URL.Query().Get("reveal") == "1"
	writeJSON(w, http.StatusOK, toFindingJSON(f, reveal))
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

var validStatuses = map[string]store.FindingStatus{
	"new":            store.StatusNew,
	"acknowledged":   store.StatusAcknowledged,
	"resolved":       store.StatusResolved,
	"false_positive": store.StatusFalsePositive,
}

func (s *Server) handleUpdateFindingStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateStatusRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	status, ok := validStatuses[req.Status]
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}

	if err := s.Store.UpdateFindingStatus(r.Context(), id, status); err != nil {
		writeError(w, http.StatusInternalServerError, "updating status")
		return
	}

	// Marking a finding false-positive also records it in the shared
	// allowlist by fingerprint, so it stays suppressed if it's ever
	// re-upserted (e.g. after a repo is removed and re-added).
	if status == store.StatusFalsePositive {
		f, err := s.Store.FindingByID(r.Context(), id)
		if err == nil {
			_, _ = s.Store.CreateAllowlistEntry(r.Context(), store.NewAllowlistEntry{
				Fingerprint: f.Fingerprint,
				Reason:      "marked false positive from dashboard",
			})
		}
	}

	writeJSON(w, http.StatusOK, nil)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	counts, err := s.Store.FindingCounts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "loading stats")
		return
	}
	repos, err := s.Store.ListRepos(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "loading stats")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"findingCounts": counts,
		"repoCount":     len(repos),
	})
}

func (s *Server) handleRecentScans(w http.ResponseWriter, r *http.Request) {
	runs, err := s.Store.RecentScanRuns(r.Context(), 50)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "loading scan runs")
		return
	}
	writeJSON(w, http.StatusOK, runs)
}

package api

import (
	"net/http"
	"strconv"

	"github.com/Laaaaksh/leakboard/internal/store"
)

type allowlistJSON struct {
	ID          int64  `json:"id"`
	RuleID      string `json:"ruleId"`
	PathPattern string `json:"pathPattern"`
	Regex       string `json:"regex"`
	Fingerprint string `json:"fingerprint"`
	Reason      string `json:"reason"`
	CreatedAt   string `json:"createdAt"`
}

func toAllowlistJSON(e store.AllowlistEntry) allowlistJSON {
	return allowlistJSON{
		ID: e.ID, RuleID: e.RuleID, PathPattern: e.PathPattern, Regex: e.Regex,
		Fingerprint: e.Fingerprint, Reason: e.Reason, CreatedAt: e.CreatedAt.Format(timeFormat),
	}
}

func (s *Server) handleListAllowlist(w http.ResponseWriter, r *http.Request) {
	entries, err := s.Store.ListAllowlistEntries(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "listing allowlist")
		return
	}
	out := make([]allowlistJSON, len(entries))
	for i, e := range entries {
		out[i] = toAllowlistJSON(e)
	}
	writeJSON(w, http.StatusOK, out)
}

type createAllowlistRequest struct {
	RuleID      string `json:"ruleId"`
	PathPattern string `json:"pathPattern"`
	Regex       string `json:"regex"`
	Reason      string `json:"reason"`
}

// handleCreateAllowlist adds an org-wide suppression rule by rule ID and/or
// path glob, or by regex matched against the secret value. Per-finding
// false positives are recorded via POST /api/findings/{id}/status instead.
func (s *Server) handleCreateAllowlist(w http.ResponseWriter, r *http.Request) {
	var req createAllowlistRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.RuleID == "" && req.PathPattern == "" && req.Regex == "" {
		writeError(w, http.StatusBadRequest, "at least one of ruleId, pathPattern, or regex is required")
		return
	}

	entry, err := s.Store.CreateAllowlistEntry(r.Context(), store.NewAllowlistEntry{
		RuleID: req.RuleID, PathPattern: req.PathPattern, Regex: req.Regex, Reason: req.Reason,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "creating allowlist entry")
		return
	}
	writeJSON(w, http.StatusCreated, toAllowlistJSON(entry))
}

func (s *Server) handleDeleteAllowlist(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.Store.DeleteAllowlistEntry(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "deleting allowlist entry")
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

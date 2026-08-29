package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Laaaaksh/leakboard/internal/store"
)

type webhookJSON struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	TargetURL string `json:"targetUrl"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"createdAt"`
}

func toWebhookJSON(w store.Webhook) webhookJSON {
	return webhookJSON{ID: w.ID, Kind: string(w.Kind), TargetURL: w.TargetURL, Enabled: w.Enabled, CreatedAt: w.CreatedAt.Format(timeFormat)}
}

func (s *Server) handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	hooks, err := s.Store.ListWebhooks(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "listing webhooks")
		return
	}
	out := make([]webhookJSON, len(hooks))
	for i, h := range hooks {
		out[i] = toWebhookJSON(h)
	}
	writeJSON(w, http.StatusOK, out)
}

type createWebhookRequest struct {
	Kind      string `json:"kind"`
	TargetURL string `json:"targetUrl"`
}

var validWebhookKinds = map[string]store.WebhookKind{
	"slack":   store.WebhookSlack,
	"discord": store.WebhookDiscord,
	"generic": store.WebhookGeneric,
}

func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	var req createWebhookRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	kind, ok := validWebhookKinds[strings.ToLower(req.Kind)]
	if !ok {
		writeError(w, http.StatusBadRequest, "kind must be one of: slack, discord, generic")
		return
	}
	if req.TargetURL == "" {
		writeError(w, http.StatusBadRequest, "targetUrl is required")
		return
	}

	hook, err := s.Store.CreateWebhook(r.Context(), kind, req.TargetURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "creating webhook")
		return
	}
	writeJSON(w, http.StatusCreated, toWebhookJSON(hook))
}

func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.Store.DeleteWebhook(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "deleting webhook")
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

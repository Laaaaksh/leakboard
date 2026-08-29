package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Laaaaksh/leakboard/internal/githubapi"
	"github.com/Laaaaksh/leakboard/internal/store"
)

type connectionJSON struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	GitHubOrg string `json:"githubOrg"`
	CreatedAt string `json:"createdAt"`
}

func toConnectionJSON(c store.Connection) connectionJSON {
	return connectionJSON{ID: c.ID, Name: c.Name, GitHubOrg: c.GitHubOrg, CreatedAt: c.CreatedAt.Format(timeFormat)}
}

func (s *Server) handleListConnections(w http.ResponseWriter, r *http.Request) {
	conns, err := s.Store.ListConnections(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "listing connections")
		return
	}
	out := make([]connectionJSON, len(conns))
	for i, c := range conns {
		out[i] = toConnectionJSON(c)
	}
	writeJSON(w, http.StatusOK, out)
}

type createConnectionRequest struct {
	Name        string `json:"name"`
	GitHubOrg   string `json:"githubOrg"`
	AccessToken string `json:"accessToken"`
}

// handleCreateConnection registers a GitHub org (name + a personal access
// token with repo read scope) and immediately syncs its repo list, so
// connecting an org is the one-step "add every repo" flow the org-wide
// onboarding promises, without a full GitHub App registration.
func (s *Server) handleCreateConnection(w http.ResponseWriter, r *http.Request) {
	var req createConnectionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.GitHubOrg = strings.TrimSpace(req.GitHubOrg)
	if req.Name == "" || req.GitHubOrg == "" || req.AccessToken == "" {
		writeError(w, http.StatusBadRequest, "name, githubOrg and accessToken are required")
		return
	}

	client := githubapi.New(req.AccessToken)
	repos, err := client.ListOrgRepos(r.Context(), req.GitHubOrg)
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not list org repos (check the org name and token scopes): "+err.Error())
		return
	}

	conn, err := s.Store.CreateConnection(r.Context(), req.Name, req.GitHubOrg, req.AccessToken)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "creating connection")
		return
	}

	added := s.syncRepos(r.Context(), conn, repos)
	writeJSON(w, http.StatusCreated, map[string]any{
		"connection": toConnectionJSON(conn),
		"reposAdded": added,
	})
}

func (s *Server) syncRepos(ctx context.Context, conn store.Connection, repos []githubapi.Repo) int {
	added := 0
	for _, r := range repos {
		_, err := s.Store.CreateRepo(ctx, store.NewRepo{
			ConnectionID:  &conn.ID,
			Name:          r.FullName,
			CloneURL:      r.CloneURL,
			DefaultBranch: r.DefaultBranch,
			MirrorPath:    mirrorPathFor(s.WorkDir, r.CloneURL),
		})
		if err == nil {
			added++
		} else {
			s.Log.Error("sync repo", "repo", r.FullName, "error", err)
		}
	}
	return added
}

func (s *Server) handleDeleteConnection(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.Store.DeleteConnection(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "deleting connection")
		return
	}
	writeJSON(w, http.StatusOK, nil)
}

// handleSyncConnection re-lists the org's repos and adds any new ones found
// since the connection was created (renamed/newly-created repos), without
// removing repos already tracked.
func (s *Server) handleSyncConnection(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	conn, err := s.Store.ConnectionByID(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "loading connection")
		return
	}

	client := githubapi.New(conn.AccessToken)
	repos, err := client.ListOrgRepos(r.Context(), conn.GitHubOrg)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not list org repos: "+err.Error())
		return
	}

	added := s.syncRepos(r.Context(), conn, repos)
	writeJSON(w, http.StatusOK, map[string]any{"reposAdded": added})
}

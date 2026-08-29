package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"testing/fstest"

	"github.com/Laaaaksh/leakboard/internal/alert"
	"github.com/Laaaaksh/leakboard/internal/gitleaks"
	"github.com/Laaaaksh/leakboard/internal/scanner"
	"github.com/Laaaaksh/leakboard/internal/store"
)

func testServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping Postgres-backed test")
	}

	db, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	for _, tbl := range []string{"webhooks", "allowlist_entries", "findings", "scan_runs", "repos", "connections", "sessions", "users"} {
		if _, err := db.DB().Exec("TRUNCATE TABLE " + tbl + " RESTART IDENTITY CASCADE"); err != nil {
			t.Fatalf("truncate %s: %v", tbl, err)
		}
	}

	srv := &Server{
		Store:   db,
		Scanner: &scanner.Scanner{Store: db, Gitleaks: gitleaks.New("gitleaks"), Alerts: alert.New(), Log: slog.New(slog.NewTextHandler(io.Discard, nil))},
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		WorkDir: t.TempDir(),
	}

	frontend := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("<html></html>")}}
	ts := httptest.NewServer(srv.Routes(frontend))
	t.Cleanup(ts.Close)
	return ts, db
}

type client struct {
	t    *testing.T
	base string
	jar  string // session cookie value, once set
}

func (c *client) do(method, path string, body any) *http.Response {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		c.t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.jar != "" {
		req.AddCookie(&http.Cookie{Name: "leakboard_session", Value: c.jar})
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatalf("do request: %v", err)
	}
	for _, ck := range resp.Cookies() {
		if ck.Name == "leakboard_session" {
			c.jar = ck.Value
		}
	}
	return resp
}

func decode[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var v T
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return v
}

func TestSetupAndLoginFlow(t *testing.T) {
	ts, _ := testServer(t)
	c := &client{t: t, base: ts.URL}

	sessionResp := c.do("GET", "/api/session", nil)
	session := decode[map[string]any](t, sessionResp)
	if session["setupRequired"] != true {
		t.Fatalf("expected setupRequired=true before any user exists, got %+v", session)
	}

	resp := c.do("POST", "/api/setup", credentials{Email: "admin@example.com", Password: "hunter22222"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("setup: got status %d", resp.StatusCode)
	}
	resp.Body.Close()

	// A second setup call must be rejected: only one admin account per instance.
	resp2 := c.do("POST", "/api/setup", credentials{Email: "other@example.com", Password: "hunter22222"})
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("second setup: got status %d, want 409", resp2.StatusCode)
	}
	resp2.Body.Close()

	// Logging out then back in should work with the same credentials.
	c.do("POST", "/api/logout", nil).Body.Close()
	loginResp := c.do("POST", "/api/login", credentials{Email: "admin@example.com", Password: "hunter22222"})
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login: got status %d", loginResp.StatusCode)
	}
	loginResp.Body.Close()

	badLogin := c.do("POST", "/api/login", credentials{Email: "admin@example.com", Password: "wrong"})
	if badLogin.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad login: got status %d, want 401", badLogin.StatusCode)
	}
	badLogin.Body.Close()
}

func TestFindingsRequireAuth(t *testing.T) {
	ts, _ := testServer(t)
	c := &client{t: t, base: ts.URL}

	resp := c.do("GET", "/api/findings", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without a session, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestFindingLifecycle_MarkingFalsePositiveCreatesAllowlistEntry(t *testing.T) {
	ts, db := testServer(t)
	c := &client{t: t, base: ts.URL}
	c.do("POST", "/api/setup", credentials{Email: "admin@example.com", Password: "hunter22222"}).Body.Close()

	repoResp := c.do("POST", "/api/repos", createRepoRequest{Name: "acme/api", CloneURL: "https://example.com/acme/api.git"})
	if repoResp.StatusCode != http.StatusCreated {
		t.Fatalf("create repo: status %d", repoResp.StatusCode)
	}
	repo := decode[repoJSON](t, repoResp)

	ctx := context.Background()
	_, isNew, err := db.UpsertFinding(ctx, store.Finding{
		RepoID: repo.ID, Fingerprint: "fp1", RuleID: "aws-access-token", FilePath: "a.js", Secret: "AKIA1234567890123456",
	}, store.StatusNew)
	if err != nil || !isNew {
		t.Fatalf("seed finding: isNew=%v err=%v", isNew, err)
	}

	listResp := c.do("GET", "/api/findings", nil)
	findings := decode[[]findingJSON](t, listResp)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].Secret == "AKIA1234567890123456" {
		t.Error("finding list should mask the secret by default")
	}

	id := findings[0].ID
	statusResp := c.do("POST", "/api/findings/"+strconv.FormatInt(id, 10)+"/status", updateStatusRequest{Status: "false_positive"})
	if statusResp.StatusCode != http.StatusOK {
		t.Fatalf("update status: %d", statusResp.StatusCode)
	}
	statusResp.Body.Close()

	entries, err := db.ListAllowlistEntries(ctx)
	if err != nil {
		t.Fatalf("ListAllowlistEntries: %v", err)
	}
	if len(entries) != 1 || entries[0].Fingerprint != "fp1" {
		t.Fatalf("expected an allowlist entry for fp1, got %+v", entries)
	}
}

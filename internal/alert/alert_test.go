package alert

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Laaaaksh/leakboard/internal/store"
)

func TestSend_FormatsPerWebhookKind(t *testing.T) {
	var received []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		received = append(received, body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	hooks := []store.Webhook{
		{ID: 1, Kind: store.WebhookSlack, TargetURL: server.URL, Enabled: true},
		{ID: 2, Kind: store.WebhookDiscord, TargetURL: server.URL, Enabled: true},
		{ID: 3, Kind: store.WebhookGeneric, TargetURL: server.URL, Enabled: true},
	}

	n := Notification{
		RepoName: "acme/api", RuleID: "aws-access-token", FilePath: "src/config.go",
		Line: 42, CommitSHA: "deadbeefcafebabe", Author: "alice", FindingURL: "https://lb.example.com/findings/1",
	}

	d := New()
	if errs := d.Send(context.Background(), hooks, n); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(received) != 3 {
		t.Fatalf("got %d webhook calls, want 3", len(received))
	}

	if _, ok := received[0]["text"].(string); !ok {
		t.Errorf("slack payload missing text field: %+v", received[0])
	}
	if _, ok := received[1]["content"].(string); !ok {
		t.Errorf("discord payload missing content field: %+v", received[1])
	}
	generic := received[2]
	if generic["rule_id"] != "aws-access-token" || generic["repo"] != "acme/api" {
		t.Errorf("generic payload missing structured fields: %+v", generic)
	}
}

func TestSend_ReportsErrorWithoutStoppingOtherWebhooks(t *testing.T) {
	var calls int
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()

	hooks := []store.Webhook{
		{ID: 1, Kind: store.WebhookGeneric, TargetURL: bad.URL, Enabled: true},
		{ID: 2, Kind: store.WebhookGeneric, TargetURL: ok.URL, Enabled: true},
	}

	d := New()
	errs := d.Send(context.Background(), hooks, Notification{RepoName: "acme/api"})
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
	if calls != 1 {
		t.Errorf("second (working) webhook should still have been called, got %d calls", calls)
	}
}

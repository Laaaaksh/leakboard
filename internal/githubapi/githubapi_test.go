package githubapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListOrgRepos_PaginatesAndDropsArchived(t *testing.T) {
	page1 := make([]Repo, pageSize)
	for i := range page1 {
		page1[i] = Repo{Name: "repo", FullName: "acme/repo"}
	}
	page2 := []Repo{
		{Name: "active", FullName: "acme/active"},
		{Name: "old", FullName: "acme/old", Archived: true},
	}

	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		if page == "1" {
			json.NewEncoder(w).Encode(page1)
			return
		}
		json.NewEncoder(w).Encode(page2)
	}))
	defer server.Close()

	c := &Client{Token: "test-token", HTTPClient: server.Client()}
	c.HTTPClient.Transport = redirectTransport{targetHost: server.Listener.Addr().String()}

	repos, err := c.ListOrgRepos(context.Background(), "acme")
	if err != nil {
		t.Fatalf("ListOrgRepos: %v", err)
	}

	if gotAuth != "Bearer test-token" {
		t.Errorf("expected bearer token auth header, got %q", gotAuth)
	}

	// pageSize repos from page 1 + 1 non-archived repo from page 2.
	if len(repos) != pageSize+1 {
		t.Fatalf("got %d repos, want %d (archived repo should be dropped)", len(repos), pageSize+1)
	}
	for _, r := range repos {
		if r.Archived {
			t.Errorf("archived repo %q leaked into results", r.FullName)
		}
	}
}

func TestAuthenticatedCloneURL(t *testing.T) {
	got := AuthenticatedCloneURL("https://github.com/acme/api.git", "tok123")
	want := "https://x-access-token:tok123@github.com/acme/api.git"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if got := AuthenticatedCloneURL("https://github.com/acme/api.git", ""); got != "https://github.com/acme/api.git" {
		t.Errorf("empty token should return the url unchanged, got %q", got)
	}
}

// redirectTransport rewrites every request to hit the test server
// regardless of the (fake) api.github.com host baked into the client.
type redirectTransport struct {
	targetHost string
}

func (t redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = t.targetHost
	return http.DefaultTransport.RoundTrip(req)
}

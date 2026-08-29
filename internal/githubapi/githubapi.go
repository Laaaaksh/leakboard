// Package githubapi is a minimal client for the one GitHub REST endpoint
// Leakboard needs: listing every repository in an organization the
// connecting token can see. It deliberately avoids a full SDK dependency.
package githubapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	apiBase  = "https://api.github.com"
	pageSize = 100
)

// Repo is the subset of GitHub's repository API response Leakboard uses.
type Repo struct {
	Name          string `json:"name"`
	FullName      string `json:"full_name"`
	CloneURL      string `json:"clone_url"`
	DefaultBranch string `json:"default_branch"`
	Private       bool   `json:"private"`
	Archived      bool   `json:"archived"`
}

// Client calls the GitHub REST API with a personal access token.
type Client struct {
	Token      string
	HTTPClient *http.Client
}

// New returns a Client authenticating with token.
func New(token string) *Client {
	return &Client{Token: token, HTTPClient: http.DefaultClient}
}

// ListOrgRepos returns every non-archived repository in org visible to the
// token, paginating through GitHub's 100-per-page results.
func (c *Client) ListOrgRepos(ctx context.Context, org string) ([]Repo, error) {
	var all []Repo
	page := 1
	for {
		repos, hasNext, err := c.listOrgReposPage(ctx, org, page)
		if err != nil {
			return nil, err
		}
		for _, r := range repos {
			if !r.Archived {
				all = append(all, r)
			}
		}
		if !hasNext {
			break
		}
		page++
	}
	return all, nil
}

func (c *Client) listOrgReposPage(ctx context.Context, org string, page int) ([]Repo, bool, error) {
	url := fmt.Sprintf("%s/orgs/%s/repos?type=all&per_page=%d&page=%d", apiBase, org, pageSize, page)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("list org repos: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("github api %s: unexpected status %s", url, resp.Status)
	}

	var repos []Repo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, false, fmt.Errorf("decode github response: %w", err)
	}

	return repos, len(repos) == pageSize, nil
}

// AuthenticatedCloneURL rewrites an HTTPS clone URL to embed the token, so
// `git clone`/`git fetch` can authenticate against a private repo without
// any credential helper configuration.
func AuthenticatedCloneURL(cloneURL, token string) string {
	const prefix = "https://"
	if token == "" || len(cloneURL) < len(prefix) || cloneURL[:len(prefix)] != prefix {
		return cloneURL
	}
	return prefix + "x-access-token:" + token + "@" + cloneURL[len(prefix):]
}

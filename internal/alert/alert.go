// Package alert sends outbound notifications when a genuinely new secret is
// found. It never re-alerts on a finding the scanner has already reported.
package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Laaaaksh/leakboard/internal/store"
)

// Notification is the information a webhook payload needs about a new finding.
type Notification struct {
	RepoName   string
	RuleID     string
	FilePath   string
	Line       int
	CommitSHA  string
	Author     string
	FindingURL string
}

// Dispatcher sends new-finding notifications to configured webhooks.
type Dispatcher struct {
	HTTPClient *http.Client
}

// New returns a Dispatcher with a sane default HTTP timeout.
func New() *Dispatcher {
	return &Dispatcher{HTTPClient: &http.Client{Timeout: 10 * time.Second}}
}

// Send delivers n to every enabled webhook, formatted for that webhook's
// kind. A failed delivery is logged by the caller and does not block the
// others.
func (d *Dispatcher) Send(ctx context.Context, hooks []store.Webhook, n Notification) []error {
	var errs []error
	for _, h := range hooks {
		if err := d.sendOne(ctx, h, n); err != nil {
			errs = append(errs, fmt.Errorf("webhook %d (%s): %w", h.ID, h.Kind, err))
		}
	}
	return errs
}

func (d *Dispatcher) sendOne(ctx context.Context, h store.Webhook, n Notification) error {
	var payload any
	switch h.Kind {
	case store.WebhookSlack:
		payload = slackPayload(n)
	case store.WebhookDiscord:
		payload = discordPayload(n)
	default:
		payload = genericPayload(n)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.TargetURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("post webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %s", resp.Status)
	}
	return nil
}

func summary(n Notification) string {
	return fmt.Sprintf("New secret found in %s: %s at %s:%d (commit %.8s, by %s)",
		n.RepoName, n.RuleID, n.FilePath, n.Line, n.CommitSHA, n.Author)
}

func slackPayload(n Notification) map[string]any {
	text := summary(n)
	if n.FindingURL != "" {
		text += "\n" + n.FindingURL
	}
	return map[string]any{"text": text}
}

func discordPayload(n Notification) map[string]any {
	content := summary(n)
	if n.FindingURL != "" {
		content += "\n" + n.FindingURL
	}
	return map[string]any{"content": content}
}

func genericPayload(n Notification) map[string]any {
	return map[string]any{
		"event":      "new_finding",
		"repo":       n.RepoName,
		"rule_id":    n.RuleID,
		"file":       n.FilePath,
		"line":       n.Line,
		"commit_sha": n.CommitSHA,
		"author":     n.Author,
		"url":        n.FindingURL,
	}
}

package store

import (
	"context"
	"fmt"
)

// CreateWebhook adds an alert destination.
func (s *Store) CreateWebhook(ctx context.Context, kind WebhookKind, targetURL string) (Webhook, error) {
	var w Webhook
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO webhooks (kind, target_url) VALUES ($1, $2)
		RETURNING id, kind, target_url, enabled, created_at
	`, kind, targetURL).Scan(&w.ID, &w.Kind, &w.TargetURL, &w.Enabled, &w.CreatedAt)
	if err != nil {
		return Webhook{}, fmt.Errorf("create webhook: %w", err)
	}
	return w, nil
}

// ListWebhooks returns every configured webhook, enabled or not.
func (s *Store) ListWebhooks(ctx context.Context) ([]Webhook, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, kind, target_url, enabled, created_at FROM webhooks ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list webhooks: %w", err)
	}
	defer rows.Close()

	var out []Webhook
	for rows.Next() {
		var w Webhook
		if err := rows.Scan(&w.ID, &w.Kind, &w.TargetURL, &w.Enabled, &w.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// EnabledWebhooks returns webhooks the scanner should notify on new findings.
func (s *Store) EnabledWebhooks(ctx context.Context) ([]Webhook, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, kind, target_url, enabled, created_at FROM webhooks WHERE enabled ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("enabled webhooks: %w", err)
	}
	defer rows.Close()

	var out []Webhook
	for rows.Next() {
		var w Webhook
		if err := rows.Scan(&w.ID, &w.Kind, &w.TargetURL, &w.Enabled, &w.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// DeleteWebhook removes an alert destination.
func (s *Store) DeleteWebhook(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM webhooks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete webhook: %w", err)
	}
	return nil
}

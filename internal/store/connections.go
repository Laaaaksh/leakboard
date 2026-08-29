package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// CreateConnection registers a GitHub org connection.
func (s *Store) CreateConnection(ctx context.Context, name, githubOrg, accessToken string) (Connection, error) {
	var c Connection
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO connections (name, github_org, access_token) VALUES ($1, $2, $3)
		 RETURNING id, name, github_org, access_token, created_at`,
		name, githubOrg, accessToken,
	).Scan(&c.ID, &c.Name, &c.GitHubOrg, &c.AccessToken, &c.CreatedAt)
	if err != nil {
		return Connection{}, fmt.Errorf("create connection: %w", err)
	}
	return c, nil
}

// ListConnections returns every connected GitHub org.
func (s *Store) ListConnections(ctx context.Context) ([]Connection, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, github_org, access_token, created_at FROM connections ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}
	defer rows.Close()

	var out []Connection
	for rows.Next() {
		var c Connection
		if err := rows.Scan(&c.ID, &c.Name, &c.GitHubOrg, &c.AccessToken, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan connection: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ConnectionByID looks up a connection, returning ErrNotFound if absent.
func (s *Store) ConnectionByID(ctx context.Context, id int64) (Connection, error) {
	var c Connection
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, github_org, access_token, created_at FROM connections WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.GitHubOrg, &c.AccessToken, &c.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Connection{}, ErrNotFound
	}
	if err != nil {
		return Connection{}, fmt.Errorf("connection by id: %w", err)
	}
	return c, nil
}

// DeleteConnection removes a connection. Repos it added stay tracked, with
// their connection_id cleared (see the ON DELETE SET NULL in migration
// 0001), since removing a connection shouldn't silently drop scan history.
func (s *Store) DeleteConnection(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM connections WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete connection: %w", err)
	}
	return nil
}

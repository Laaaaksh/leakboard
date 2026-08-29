package store

import (
	"context"
	"database/sql"
	"fmt"
)

// NewAllowlistEntry is the input to CreateAllowlistEntry.
type NewAllowlistEntry struct {
	RuleID      string
	PathPattern string
	Regex       string
	Fingerprint string
	Reason      string
}

// CreateAllowlistEntry adds a suppression rule.
func (s *Store) CreateAllowlistEntry(ctx context.Context, e NewAllowlistEntry) (AllowlistEntry, error) {
	var out AllowlistEntry
	var ruleID, path, regex, fp sql.NullString
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO allowlist_entries (rule_id, path_pattern, regex, fingerprint, reason)
		VALUES (NULLIF($1,''), NULLIF($2,''), NULLIF($3,''), NULLIF($4,''), $5)
		RETURNING id, rule_id, path_pattern, regex, fingerprint, reason, created_at
	`, e.RuleID, e.PathPattern, e.Regex, e.Fingerprint, e.Reason,
	).Scan(&out.ID, &ruleID, &path, &regex, &fp, &out.Reason, &out.CreatedAt)
	if err != nil {
		return AllowlistEntry{}, fmt.Errorf("create allowlist entry: %w", err)
	}
	out.RuleID, out.PathPattern, out.Regex, out.Fingerprint = ruleID.String, path.String, regex.String, fp.String
	return out, nil
}

// ListAllowlistEntries returns every suppression rule, newest first.
func (s *Store) ListAllowlistEntries(ctx context.Context) ([]AllowlistEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, rule_id, path_pattern, regex, fingerprint, reason, created_at
		FROM allowlist_entries ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list allowlist entries: %w", err)
	}
	defer rows.Close()

	var out []AllowlistEntry
	for rows.Next() {
		var e AllowlistEntry
		var ruleID, path, regex, fp sql.NullString
		if err := rows.Scan(&e.ID, &ruleID, &path, &regex, &fp, &e.Reason, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan allowlist entry: %w", err)
		}
		e.RuleID, e.PathPattern, e.Regex, e.Fingerprint = ruleID.String, path.String, regex.String, fp.String
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteAllowlistEntry removes a suppression rule.
func (s *Store) DeleteAllowlistEntry(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM allowlist_entries WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete allowlist entry: %w", err)
	}
	return nil
}

package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// UpsertFinding records a detected secret. If a finding with the same
// (repo_id, fingerprint) already exists, only its last_seen_at is bumped and
// isNew is false; a scheduled rescan of already-known findings must never
// re-alert. Otherwise a new row is inserted with the given status (StatusNew
// unless the fingerprint is allowlisted, in which case the caller passes
// StatusFalsePositive so it never surfaces as "new") and isNew reports true.
func (s *Store) UpsertFinding(ctx context.Context, f Finding, initialStatus FindingStatus) (id int64, isNew bool, err error) {
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO findings (repo_id, fingerprint, rule_id, description, file_path, start_line, end_line,
			commit_sha, commit_author, commit_email, commit_date, secret, match, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (repo_id, fingerprint) DO UPDATE SET last_seen_at = now()
		RETURNING id, (xmax = 0) AS inserted
	`, f.RepoID, f.Fingerprint, f.RuleID, f.Description, f.FilePath, f.StartLine, f.EndLine,
		f.CommitSHA, f.CommitAuthor, f.CommitEmail, f.CommitDate, f.Secret, f.Match, initialStatus,
	).Scan(&id, &isNew)
	if err != nil {
		return 0, false, fmt.Errorf("upsert finding: %w", err)
	}
	return id, isNew, nil
}

// FindingFilter narrows ListFindings; a zero value matches everything.
type FindingFilter struct {
	RepoID *int64
	Status FindingStatus
	RuleID string
}

// ListFindings returns findings matching f, newest first.
func (s *Store) ListFindings(ctx context.Context, f FindingFilter) ([]Finding, error) {
	q := `SELECT f.id, f.repo_id, r.name, f.fingerprint, f.rule_id, f.description, f.file_path,
		f.start_line, f.end_line, f.commit_sha, f.commit_author, f.commit_email, f.commit_date,
		f.secret, f.match, f.status, f.first_seen_at, f.last_seen_at, f.resolved_at
		FROM findings f JOIN repos r ON r.id = f.repo_id WHERE 1=1`
	var args []any
	if f.RepoID != nil {
		args = append(args, *f.RepoID)
		q += fmt.Sprintf(" AND f.repo_id = $%d", len(args))
	}
	if f.Status != "" {
		args = append(args, f.Status)
		q += fmt.Sprintf(" AND f.status = $%d", len(args))
	}
	if f.RuleID != "" {
		args = append(args, f.RuleID)
		q += fmt.Sprintf(" AND f.rule_id = $%d", len(args))
	}
	q += " ORDER BY f.first_seen_at DESC"

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list findings: %w", err)
	}
	defer rows.Close()

	var out []Finding
	for rows.Next() {
		fnd, err := scanFinding(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, fnd)
	}
	return out, rows.Err()
}

// FindingByID looks up a finding, returning ErrNotFound if absent.
func (s *Store) FindingByID(ctx context.Context, id int64) (Finding, error) {
	row := s.db.QueryRowContext(ctx, `SELECT f.id, f.repo_id, r.name, f.fingerprint, f.rule_id, f.description, f.file_path,
		f.start_line, f.end_line, f.commit_sha, f.commit_author, f.commit_email, f.commit_date,
		f.secret, f.match, f.status, f.first_seen_at, f.last_seen_at, f.resolved_at
		FROM findings f JOIN repos r ON r.id = f.repo_id WHERE f.id = $1`, id)
	fnd, err := scanFinding(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Finding{}, ErrNotFound
	}
	return fnd, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanFinding(row rowScanner) (Finding, error) {
	var f Finding
	var desc, author, email, match sql.NullString
	var commitDate sql.NullTime
	var resolvedAt sql.NullTime
	err := row.Scan(&f.ID, &f.RepoID, &f.RepoName, &f.Fingerprint, &f.RuleID, &desc, &f.FilePath,
		&f.StartLine, &f.EndLine, &f.CommitSHA, &author, &email, &commitDate,
		&f.Secret, &match, &f.Status, &f.FirstSeenAt, &f.LastSeenAt, &resolvedAt)
	if err != nil {
		return Finding{}, fmt.Errorf("scan finding: %w", err)
	}
	f.Description = desc.String
	f.CommitAuthor = author.String
	f.CommitEmail = email.String
	f.Match = match.String
	if commitDate.Valid {
		f.CommitDate = &commitDate.Time
	}
	if resolvedAt.Valid {
		f.ResolvedAt = &resolvedAt.Time
	}
	return f, nil
}

// UpdateFindingStatus transitions a finding's status, stamping ResolvedAt
// when it moves to resolved or false_positive.
func (s *Store) UpdateFindingStatus(ctx context.Context, id int64, status FindingStatus) error {
	var resolvedAt any
	if status == StatusResolved || status == StatusFalsePositive {
		resolvedAt = time.Now()
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE findings SET status = $2, resolved_at = $3 WHERE id = $1`, id, status, resolvedAt)
	if err != nil {
		return fmt.Errorf("update finding status: %w", err)
	}
	return nil
}

// FindingCounts returns the count of findings per status, for dashboard
// summary tiles.
func (s *Store) FindingCounts(ctx context.Context) (map[FindingStatus]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT status, count(*) FROM findings GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("finding counts: %w", err)
	}
	defer rows.Close()

	out := map[FindingStatus]int{}
	for rows.Next() {
		var status FindingStatus
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			return nil, fmt.Errorf("scan finding count: %w", err)
		}
		out[status] = n
	}
	return out, rows.Err()
}

// FingerprintsMatchingAllowlist returns which of the given fingerprints are
// covered by an existing fingerprint-based allowlist entry.
func (s *Store) FingerprintsMatchingAllowlist(ctx context.Context, fingerprints []string) (map[string]bool, error) {
	if len(fingerprints) == 0 {
		return map[string]bool{}, nil
	}
	placeholders := make([]string, len(fingerprints))
	args := make([]any, len(fingerprints))
	for i, fp := range fingerprints {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = fp
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT fingerprint FROM allowlist_entries WHERE fingerprint IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("fingerprints matching allowlist: %w", err)
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var fp string
		if err := rows.Scan(&fp); err != nil {
			return nil, fmt.Errorf("scan fingerprint: %w", err)
		}
		out[fp] = true
	}
	return out, rows.Err()
}

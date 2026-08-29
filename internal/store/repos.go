package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// NewRepo is the input to CreateRepo.
type NewRepo struct {
	ConnectionID  *int64
	Name          string
	CloneURL      string
	DefaultBranch string
	MirrorPath    string
}

// CreateRepo inserts a repo to track, or returns the existing row if one
// with the same CloneURL is already tracked (ON CONFLICT DO NOTHING + a
// follow-up read), so re-syncing an org's repo list is idempotent.
func (s *Store) CreateRepo(ctx context.Context, r NewRepo) (Repo, error) {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO repos (connection_id, name, clone_url, default_branch, mirror_path)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (clone_url) DO NOTHING
	`, r.ConnectionID, r.Name, r.CloneURL, r.DefaultBranch, r.MirrorPath)
	if err != nil {
		return Repo{}, fmt.Errorf("create repo: %w", err)
	}
	return s.RepoByCloneURL(ctx, r.CloneURL)
}

// RepoByCloneURL looks up a repo by its clone URL, returning ErrNotFound if absent.
func (s *Store) RepoByCloneURL(ctx context.Context, cloneURL string) (Repo, error) {
	return s.scanRepoRow(s.db.QueryRowContext(ctx, repoSelect+` WHERE clone_url = $1`, cloneURL))
}

// RepoByID looks up a repo, returning ErrNotFound if absent.
func (s *Store) RepoByID(ctx context.Context, id int64) (Repo, error) {
	return s.scanRepoRow(s.db.QueryRowContext(ctx, repoSelect+` WHERE id = $1`, id))
}

const repoSelect = `SELECT id, connection_id, name, clone_url, default_branch, mirror_path,
	scanned_ref_tips, last_scanned_at, last_scan_error, scan_interval_secs, created_at FROM repos`

func (s *Store) scanRepoRow(row *sql.Row) (Repo, error) {
	var r Repo
	var refTips []byte
	err := row.Scan(&r.ID, &r.ConnectionID, &r.Name, &r.CloneURL, &r.DefaultBranch, &r.MirrorPath,
		&refTips, &r.LastScannedAt, &r.LastScanError, &r.ScanIntervalSecs, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Repo{}, ErrNotFound
	}
	if err != nil {
		return Repo{}, fmt.Errorf("scan repo: %w", err)
	}
	if err := json.Unmarshal(refTips, &r.ScannedRefTips); err != nil {
		return Repo{}, fmt.Errorf("decode ref tips: %w", err)
	}
	return r, nil
}

// ListRepos returns every tracked repo, alphabetically by name.
func (s *Store) ListRepos(ctx context.Context) ([]Repo, error) {
	rows, err := s.db.QueryContext(ctx, repoSelect+` ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}
	defer rows.Close()

	var out []Repo
	for rows.Next() {
		var r Repo
		var refTips []byte
		if err := rows.Scan(&r.ID, &r.ConnectionID, &r.Name, &r.CloneURL, &r.DefaultBranch, &r.MirrorPath,
			&refTips, &r.LastScannedAt, &r.LastScanError, &r.ScanIntervalSecs, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan repo: %w", err)
		}
		if err := json.Unmarshal(refTips, &r.ScannedRefTips); err != nil {
			return nil, fmt.Errorf("decode ref tips: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ReposDueForScan returns repos whose last scan (if any) is older than their
// configured interval.
func (s *Store) ReposDueForScan(ctx context.Context, now time.Time) ([]Repo, error) {
	rows, err := s.db.QueryContext(ctx, repoSelect+`
		WHERE last_scanned_at IS NULL
		   OR last_scanned_at < $1::timestamptz - (scan_interval_secs * interval '1 second')
		ORDER BY last_scanned_at NULLS FIRST`, now)
	if err != nil {
		return nil, fmt.Errorf("repos due for scan: %w", err)
	}
	defer rows.Close()

	var out []Repo
	for rows.Next() {
		var r Repo
		var refTips []byte
		if err := rows.Scan(&r.ID, &r.ConnectionID, &r.Name, &r.CloneURL, &r.DefaultBranch, &r.MirrorPath,
			&refTips, &r.LastScannedAt, &r.LastScanError, &r.ScanIntervalSecs, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan repo: %w", err)
		}
		if err := json.Unmarshal(refTips, &r.ScannedRefTips); err != nil {
			return nil, fmt.Errorf("decode ref tips: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpdateScanResult records the outcome of a completed scan attempt: the new
// ref tips to diff against next time (nil leaves them unchanged, used on
// failure), and any error message (empty clears a prior one).
func (s *Store) UpdateScanResult(ctx context.Context, repoID int64, refTips map[string]string, scanErr string) error {
	if refTips == nil {
		_, err := s.db.ExecContext(ctx,
			`UPDATE repos SET last_scanned_at = now(), last_scan_error = $2 WHERE id = $1`,
			repoID, scanErr)
		if err != nil {
			return fmt.Errorf("update scan result: %w", err)
		}
		return nil
	}

	encoded, err := json.Marshal(refTips)
	if err != nil {
		return fmt.Errorf("encode ref tips: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE repos SET last_scanned_at = now(), last_scan_error = $2, scanned_ref_tips = $3 WHERE id = $1`,
		repoID, scanErr, encoded)
	if err != nil {
		return fmt.Errorf("update scan result: %w", err)
	}
	return nil
}

// DeleteRepo stops tracking a repo and cascades its findings and scan runs.
func (s *Store) DeleteRepo(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM repos WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete repo: %w", err)
	}
	return nil
}

// CreateScanRun starts recording a scan attempt and returns its id.
func (s *Store) CreateScanRun(ctx context.Context, repoID int64) (int64, error) {
	var id int64
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO scan_runs (repo_id) VALUES ($1) RETURNING id`, repoID,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create scan run: %w", err)
	}
	return id, nil
}

// FinishScanRun records a scan attempt's outcome.
func (s *Store) FinishScanRun(ctx context.Context, id int64, status string, newFindings int, errMsg string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE scan_runs SET finished_at = now(), status = $2, new_findings = $3, error = $4 WHERE id = $1
	`, id, status, newFindings, errMsg)
	if err != nil {
		return fmt.Errorf("finish scan run: %w", err)
	}
	return nil
}

// RecentScanRuns returns the most recent scan attempts across all repos.
func (s *Store) RecentScanRuns(ctx context.Context, limit int) ([]ScanRun, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, repo_id, started_at, finished_at, status, new_findings, error
		FROM scan_runs ORDER BY started_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("recent scan runs: %w", err)
	}
	defer rows.Close()

	var out []ScanRun
	for rows.Next() {
		var r ScanRun
		var errMsg sql.NullString
		if err := rows.Scan(&r.ID, &r.RepoID, &r.StartedAt, &r.FinishedAt, &r.Status, &r.NewFindings, &errMsg); err != nil {
			return nil, fmt.Errorf("scan run: %w", err)
		}
		r.Error = errMsg.String
		out = append(out, r)
	}
	return out, rows.Err()
}

// Package scanner orchestrates a scan of one repo: sync the mirror clone,
// run gitleaks over commits introduced since the last scan, filter out
// allowlisted results, persist genuinely new findings, and alert on them.
package scanner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Laaaaksh/leakboard/internal/alert"
	"github.com/Laaaaksh/leakboard/internal/githubapi"
	"github.com/Laaaaksh/leakboard/internal/gitleaks"
	"github.com/Laaaaksh/leakboard/internal/gitutil"
	"github.com/Laaaaksh/leakboard/internal/store"
)

// Scanner scans repos and dispatches alerts for newly discovered findings.
type Scanner struct {
	Store    *store.Store
	Gitleaks *gitleaks.Scanner
	Alerts   *alert.Dispatcher
	WorkDir  string
	Log      *slog.Logger
	BaseURL  string
}

// ScanRepo performs one scan pass over a single repo and returns the number
// of genuinely new (never-seen-before) findings.
func (s *Scanner) ScanRepo(ctx context.Context, repo store.Repo) (newCount int, err error) {
	runID, err := s.Store.CreateScanRun(ctx, repo.ID)
	if err != nil {
		return 0, fmt.Errorf("create scan run: %w", err)
	}

	newCount, scanErr := s.scanRepoInner(ctx, repo)

	status := "success"
	errMsg := ""
	if scanErr != nil {
		status = "failed"
		errMsg = scanErr.Error()
		// scanRepoInner only records a successful result; on failure the
		// repo's last_scanned_at/error still need updating here so a
		// broken repo doesn't get retried on every scheduler tick.
		if err := s.Store.UpdateScanResult(ctx, repo.ID, nil, errMsg); err != nil {
			s.Log.Error("update scan result after failure", "repo", repo.Name, "error", err)
		}
	}
	if err := s.Store.FinishScanRun(ctx, runID, status, newCount, errMsg); err != nil {
		s.Log.Error("finish scan run", "repo", repo.Name, "error", err)
	}

	return newCount, scanErr
}

func (s *Scanner) scanRepoInner(ctx context.Context, repo store.Repo) (int, error) {
	cloneURL, err := s.authenticatedCloneURL(ctx, repo)
	if err != nil {
		return 0, fmt.Errorf("resolve credentials: %w", err)
	}
	if err := gitutil.EnsureMirror(ctx, cloneURL, repo.MirrorPath); err != nil {
		return 0, fmt.Errorf("sync mirror: %w", err)
	}

	logOpts := gitutil.BuildIncrementalLogOpts(ctx, repo.MirrorPath, repo.ScannedRefTips)

	findings, err := s.Gitleaks.Scan(ctx, repo.MirrorPath, logOpts)
	if err != nil {
		return 0, fmt.Errorf("gitleaks scan: %w", err)
	}

	allowlist, err := s.Store.ListAllowlistEntries(ctx)
	if err != nil {
		return 0, fmt.Errorf("load allowlist: %w", err)
	}
	findings = FilterAllowlisted(findings, allowlist)

	newTips, err := gitutil.RefTips(ctx, repo.MirrorPath)
	if err != nil {
		return 0, fmt.Errorf("read ref tips: %w", err)
	}

	newCount := 0
	var toAlert []store.Finding
	for _, f := range findings {
		commitDate := f.Date
		sf := store.Finding{
			RepoID:       repo.ID,
			Fingerprint:  f.Fingerprint,
			RuleID:       f.RuleID,
			Description:  f.Description,
			FilePath:     f.File,
			StartLine:    f.StartLine,
			EndLine:      f.EndLine,
			CommitSHA:    f.Commit,
			CommitAuthor: f.Author,
			CommitEmail:  f.Email,
			CommitDate:   &commitDate,
			Secret:       f.Secret,
			Match:        f.Match,
		}

		id, isNew, err := s.Store.UpsertFinding(ctx, sf, store.StatusNew)
		if err != nil {
			return newCount, fmt.Errorf("upsert finding %s: %w", f.Fingerprint, err)
		}
		if isNew {
			newCount++
			sf.ID = id
			sf.RepoName = repo.Name
			toAlert = append(toAlert, sf)
		}
	}

	if err := s.Store.UpdateScanResult(ctx, repo.ID, newTips, ""); err != nil {
		return newCount, fmt.Errorf("record scan result: %w", err)
	}

	if len(toAlert) > 0 {
		s.notify(ctx, toAlert)
	}

	return newCount, nil
}

// authenticatedCloneURL returns repo.CloneURL as-is for a manually added
// repo, or with the owning connection's access token embedded for a repo
// synced from a GitHub org, so the token lives only in the connections
// table and briefly in a git remote URL, never persisted alongside the repo.
func (s *Scanner) authenticatedCloneURL(ctx context.Context, repo store.Repo) (string, error) {
	if repo.ConnectionID == nil {
		return repo.CloneURL, nil
	}
	conn, err := s.Store.ConnectionByID(ctx, *repo.ConnectionID)
	if err != nil {
		return "", fmt.Errorf("load connection: %w", err)
	}
	return githubapi.AuthenticatedCloneURL(repo.CloneURL, conn.AccessToken), nil
}

func (s *Scanner) notify(ctx context.Context, findings []store.Finding) {
	hooks, err := s.Store.EnabledWebhooks(ctx)
	if err != nil {
		s.Log.Error("load webhooks", "error", err)
		return
	}
	if len(hooks) == 0 {
		return
	}

	for _, f := range findings {
		n := alert.Notification{
			RepoName:   f.RepoName,
			RuleID:     f.RuleID,
			FilePath:   f.FilePath,
			Line:       f.StartLine,
			CommitSHA:  f.CommitSHA,
			Author:     f.CommitAuthor,
			FindingURL: fmt.Sprintf("%s/findings/%d", s.BaseURL, f.ID),
		}
		for _, err := range s.Alerts.Send(ctx, hooks, n) {
			s.Log.Error("send alert", "error", err)
		}
	}
}

// RunDue scans every repo whose interval has elapsed. Errors scanning one
// repo are logged and do not stop the others.
func (s *Scanner) RunDue(ctx context.Context) {
	repos, err := s.Store.ReposDueForScan(ctx, time.Now())
	if err != nil {
		s.Log.Error("list repos due for scan", "error", err)
		return
	}
	for _, repo := range repos {
		n, err := s.ScanRepo(ctx, repo)
		if err != nil {
			s.Log.Error("scan repo", "repo", repo.Name, "error", err)
			continue
		}
		s.Log.Info("scanned repo", "repo", repo.Name, "new_findings", n)
	}
}

// Run starts the scheduler loop, checking for due repos every interval
// until ctx is cancelled.
func (s *Scanner) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.RunDue(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.RunDue(ctx)
		}
	}
}

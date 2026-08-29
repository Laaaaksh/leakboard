package store

import (
	"context"
	"os"
	"testing"
	"time"
)

// testStore opens a fresh Store against TEST_DATABASE_URL, migrated and
// truncated, so each test starts from a clean slate. Tests are skipped
// (not failed) when the env var is unset, since Postgres isn't available in
// every environment this repo's tests run in (see CONTRIBUTING.md).
func testStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping Postgres-backed test")
	}

	s, err := Open(context.Background(), url)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	tables := []string{"webhooks", "allowlist_entries", "findings", "scan_runs", "repos", "connections", "sessions", "users"}
	for _, tbl := range tables {
		if _, err := s.db.Exec("TRUNCATE TABLE " + tbl + " RESTART IDENTITY CASCADE"); err != nil {
			t.Fatalf("truncate %s: %v", tbl, err)
		}
	}
	return s
}

func TestUpsertFinding_DedupesByFingerprint(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	repo, err := s.CreateRepo(ctx, NewRepo{Name: "acme/api", CloneURL: "https://example.com/acme/api.git", DefaultBranch: "main", MirrorPath: "/tmp/x"})
	if err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}

	f := Finding{RepoID: repo.ID, Fingerprint: "abc:file.js:aws-access-token:1", RuleID: "aws-access-token", FilePath: "file.js", Secret: "AKIA..."}

	id1, isNew1, err := s.UpsertFinding(ctx, f, StatusNew)
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if !isNew1 {
		t.Error("first upsert of a fingerprint should report isNew=true")
	}

	id2, isNew2, err := s.UpsertFinding(ctx, f, StatusNew)
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if isNew2 {
		t.Error("second upsert of the same fingerprint should report isNew=false (must not re-alert)")
	}
	if id1 != id2 {
		t.Errorf("expected the same finding id on re-upsert, got %d and %d", id1, id2)
	}

	findings, err := s.ListFindings(ctx, FindingFilter{})
	if err != nil {
		t.Fatalf("ListFindings: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected exactly one stored finding after two upserts of the same fingerprint, got %d", len(findings))
	}
}

func TestUpsertFinding_AllowlistedInitialStatusNeverCountsAsNewAlert(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	repo, err := s.CreateRepo(ctx, NewRepo{Name: "acme/api", CloneURL: "https://example.com/acme/api2.git", DefaultBranch: "main", MirrorPath: "/tmp/y"})
	if err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}

	f := Finding{RepoID: repo.ID, Fingerprint: "abc:file.js:aws-access-token:1", RuleID: "aws-access-token", FilePath: "file.js", Secret: "AKIA..."}
	_, isNew, err := s.UpsertFinding(ctx, f, StatusFalsePositive)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if !isNew {
		t.Fatal("first insert should still report isNew=true (it's a new row)")
	}

	got, err := s.FindingByID(ctx, mustFirstID(t, ctx, s))
	if err != nil {
		t.Fatalf("FindingByID: %v", err)
	}
	if got.Status != StatusFalsePositive {
		t.Errorf("expected status false_positive, got %s", got.Status)
	}
}

func mustFirstID(t *testing.T, ctx context.Context, s *Store) int64 {
	t.Helper()
	findings, err := s.ListFindings(ctx, FindingFilter{})
	if err != nil || len(findings) == 0 {
		t.Fatalf("expected at least one finding, err=%v", err)
	}
	return findings[0].ID
}

func TestReposDueForScan(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	repo, err := s.CreateRepo(ctx, NewRepo{Name: "acme/api", CloneURL: "https://example.com/acme/due.git", DefaultBranch: "main", MirrorPath: "/tmp/z"})
	if err != nil {
		t.Fatalf("CreateRepo: %v", err)
	}

	due, err := s.ReposDueForScan(ctx, time.Now())
	if err != nil {
		t.Fatalf("ReposDueForScan: %v", err)
	}
	if len(due) != 1 || due[0].ID != repo.ID {
		t.Fatalf("expected the never-scanned repo to be due, got %+v", due)
	}

	if err := s.UpdateScanResult(ctx, repo.ID, map[string]string{"refs/heads/main": "abc123"}, ""); err != nil {
		t.Fatalf("UpdateScanResult: %v", err)
	}

	due, err = s.ReposDueForScan(ctx, time.Now())
	if err != nil {
		t.Fatalf("ReposDueForScan: %v", err)
	}
	if len(due) != 0 {
		t.Fatalf("expected no repos due immediately after a scan, got %+v", due)
	}

	future := time.Now().Add(time.Duration(repo.ScanIntervalSecs+60) * time.Second)
	due, err = s.ReposDueForScan(ctx, future)
	if err != nil {
		t.Fatalf("ReposDueForScan: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("expected the repo to be due again once its interval elapsed, got %+v", due)
	}
}

func TestFingerprintsMatchingAllowlist(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	if _, err := s.CreateAllowlistEntry(ctx, NewAllowlistEntry{Fingerprint: "known-fp", Reason: "test"}); err != nil {
		t.Fatalf("CreateAllowlistEntry: %v", err)
	}

	got, err := s.FingerprintsMatchingAllowlist(ctx, []string{"known-fp", "unknown-fp"})
	if err != nil {
		t.Fatalf("FingerprintsMatchingAllowlist: %v", err)
	}
	if !got["known-fp"] || got["unknown-fp"] {
		t.Errorf("got %+v", got)
	}
}

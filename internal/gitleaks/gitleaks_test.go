package gitleaks

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestScan_DetectsSecretAndFingerprint(t *testing.T) {
	ctx := context.Background()
	repo := t.TempDir()
	runGit(t, repo, "init", "--quiet", "--initial-branch=main")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test")

	writeFile(t, repo, "config.js", `const key = "AKIAABCDEFGHIJKLMNOP";`)
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "--quiet", "-m", "add key")

	s := New("gitleaks")
	if err := s.CheckAvailable(ctx); err != nil {
		t.Skipf("gitleaks binary not available: %v", err)
	}

	findings, err := s.Scan(ctx, repo, "")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1: %+v", len(findings), findings)
	}
	f := findings[0]
	if f.RuleID != "aws-access-token" {
		t.Errorf("got rule %q, want aws-access-token", f.RuleID)
	}
	if f.Secret != "AKIAABCDEFGHIJKLMNOP" {
		t.Errorf("got secret %q", f.Secret)
	}
	if f.Fingerprint == "" {
		t.Error("expected a non-empty fingerprint")
	}
}

func TestScan_NoSecretsReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	repo := t.TempDir()
	runGit(t, repo, "init", "--quiet", "--initial-branch=main")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test")

	writeFile(t, repo, "readme.md", "nothing sensitive here")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "--quiet", "-m", "add readme")

	s := New("gitleaks")
	if err := s.CheckAvailable(ctx); err != nil {
		t.Skipf("gitleaks binary not available: %v", err)
	}

	findings, err := s.Scan(ctx, repo, "")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("got %d findings, want 0: %+v", len(findings), findings)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out.String())
	}
}

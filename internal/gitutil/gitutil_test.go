package gitutil

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBuildIncrementalLogOpts_ExcludesKnownCommits(t *testing.T) {
	ctx := context.Background()
	src := initTestRepo(t)
	writeAndCommit(t, src, "a.txt", "first", "commit one")

	mirror := filepath.Join(t.TempDir(), "mirror.git")
	if err := EnsureMirror(ctx, src, mirror); err != nil {
		t.Fatalf("EnsureMirror: %v", err)
	}

	// No prior tips: baseline scan should be a full scan (empty log-opts).
	if got := BuildIncrementalLogOpts(ctx, mirror, nil); got != "" {
		t.Fatalf("expected empty log-opts with no prior tips, got %q", got)
	}

	tips, err := RefTips(ctx, mirror)
	if err != nil {
		t.Fatalf("RefTips: %v", err)
	}
	if len(tips) == 0 {
		t.Fatal("expected at least one ref tip after baseline clone")
	}

	writeAndCommit(t, src, "b.txt", "second", "commit two")
	if err := EnsureMirror(ctx, src, mirror); err != nil {
		t.Fatalf("EnsureMirror (fetch): %v", err)
	}

	logOpts := BuildIncrementalLogOpts(ctx, mirror, tips)
	if logOpts == "" {
		t.Fatal("expected non-empty log-opts once a baseline exists")
	}

	newTips, err := RefTips(ctx, mirror)
	if err != nil {
		t.Fatalf("RefTips after second commit: %v", err)
	}
	for ref, sha := range tips {
		if newTips[ref] == sha {
			t.Fatalf("expected ref %s to move past baseline commit %s", ref, sha)
		}
	}
}

func TestBuildIncrementalLogOpts_FallsBackWhenTipIsGone(t *testing.T) {
	ctx := context.Background()
	src := initTestRepo(t)
	writeAndCommit(t, src, "a.txt", "first", "commit one")

	mirror := filepath.Join(t.TempDir(), "mirror.git")
	if err := EnsureMirror(ctx, src, mirror); err != nil {
		t.Fatalf("EnsureMirror: %v", err)
	}

	staleTips := map[string]string{"refs/heads/main": "0000000000000000000000000000000000000000"}
	got := BuildIncrementalLogOpts(ctx, mirror, staleTips)
	if got != "--full-history --all --diff-filter=tuxdb" {
		t.Fatalf("expected the stale sha to be dropped from the exclusion list, got %q", got)
	}
}

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "--quiet", "--initial-branch=main")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	return dir
}

func writeAndCommit(t *testing.T, repoDir, file, content, message string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repoDir, file), []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "--quiet", "-m", message)
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

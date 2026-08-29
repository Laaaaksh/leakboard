package scanner

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Laaaaksh/leakboard/internal/gitleaks"
	"github.com/Laaaaksh/leakboard/internal/gitutil"
)

// TestIncrementalScan_OnlyReportsNewCommits exercises the real gitutil +
// gitleaks pipeline end to end: a baseline scan must find a secret
// committed before the baseline, and a follow-up scan using the resulting
// ref tips must find only a secret introduced afterward, never re-reporting
// the first one. This is the property the whole "don't re-alert on
// already-known findings" feature depends on.
func TestIncrementalScan_OnlyReportsNewCommits(t *testing.T) {
	ctx := context.Background()
	gl := gitleaks.New("gitleaks")
	if err := gl.CheckAvailable(ctx); err != nil {
		t.Skipf("gitleaks binary not available: %v", err)
	}

	src := t.TempDir()
	runGit(t, src, "init", "--quiet", "--initial-branch=main")
	runGit(t, src, "config", "user.email", "test@example.com")
	runGit(t, src, "config", "user.name", "Test")
	writeFile(t, src, "old.js", `const key = "AKIAABCDEFGHIJKLMNOP";`)
	runGit(t, src, "add", ".")
	runGit(t, src, "commit", "--quiet", "-m", "old secret")

	mirror := filepath.Join(t.TempDir(), "mirror.git")
	if err := gitutil.EnsureMirror(ctx, src, mirror); err != nil {
		t.Fatalf("EnsureMirror: %v", err)
	}

	baseline, err := gl.Scan(ctx, mirror, gitutil.BuildIncrementalLogOpts(ctx, mirror, nil))
	if err != nil {
		t.Fatalf("baseline scan: %v", err)
	}
	if len(baseline) != 1 || baseline[0].Secret != "AKIAABCDEFGHIJKLMNOP" {
		t.Fatalf("baseline scan should find the old secret exactly once, got %+v", baseline)
	}

	tips, err := gitutil.RefTips(ctx, mirror)
	if err != nil {
		t.Fatalf("RefTips: %v", err)
	}

	writeFile(t, src, "new.js", `const other = "AIzaSyDaGmWKa4JsXZ-HjGw7ISLan_MEK6NUn-w";`)
	runGit(t, src, "add", ".")
	runGit(t, src, "commit", "--quiet", "-m", "new secret")

	if err := gitutil.EnsureMirror(ctx, src, mirror); err != nil {
		t.Fatalf("EnsureMirror (fetch): %v", err)
	}

	incremental, err := gl.Scan(ctx, mirror, gitutil.BuildIncrementalLogOpts(ctx, mirror, tips))
	if err != nil {
		t.Fatalf("incremental scan: %v", err)
	}
	if len(incremental) != 1 {
		t.Fatalf("incremental scan should find exactly the new secret, got %d findings: %+v", len(incremental), incremental)
	}
	if incremental[0].Fingerprint == baseline[0].Fingerprint {
		t.Fatalf("incremental scan re-reported the already-known finding %q", baseline[0].Fingerprint)
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

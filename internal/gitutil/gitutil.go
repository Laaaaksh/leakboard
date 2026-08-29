// Package gitutil manages local bare mirror clones of tracked repositories
// and computes the git log range that covers only commits introduced since
// a repo's last successful scan, across every ref (not just the default
// branch).
package gitutil

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// EnsureMirror makes sure a bare mirror clone of cloneURL exists at path,
// cloning it if absent and fetching updates otherwise. A mirror clone (as
// opposed to a normal clone) tracks every ref the remote has, which is what
// makes whole-history, all-branch scanning possible.
func EnsureMirror(ctx context.Context, cloneURL, path string) error {
	if _, err := os.Stat(path); err == nil {
		return fetch(ctx, path)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat mirror path: %w", err)
	}

	if err := os.MkdirAll(parentDir(path), 0o755); err != nil {
		return fmt.Errorf("create mirror parent dir: %w", err)
	}
	return run(ctx, "", "clone", "--mirror", "--quiet", cloneURL, path)
}

func fetch(ctx context.Context, mirrorPath string) error {
	return run(ctx, "", "-C", mirrorPath, "fetch", "--prune", "--quiet", "origin")
}

func parentDir(path string) string {
	i := strings.LastIndex(path, "/")
	if i < 0 {
		return "."
	}
	return path[:i]
}

// RefTips returns every ref in the mirror and the commit SHA it currently
// points at.
func RefTips(ctx context.Context, mirrorPath string) (map[string]string, error) {
	out, err := output(ctx, "-C", mirrorPath, "for-each-ref", "--format=%(objectname) %(refname)")
	if err != nil {
		return nil, fmt.Errorf("for-each-ref: %w", err)
	}

	tips := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		tips[parts[1]] = parts[0]
	}
	return tips, nil
}

// ObjectExists reports whether sha is a valid, present object in the mirror.
func ObjectExists(ctx context.Context, mirrorPath, sha string) bool {
	cmd := exec.CommandContext(ctx, "git", "-C", mirrorPath, "cat-file", "-e", sha)
	return cmd.Run() == nil
}

// BuildIncrementalLogOpts builds a gitleaks --log-opts value that scans
// every ref in the mirror while excluding history already reachable from
// priorTips, so a rescan only sees commits introduced since the last run.
// Any priorTips SHA no longer present in the mirror (pruned after a
// force-push, say) is silently dropped from the exclusion set rather than
// failing the whole scan.
func BuildIncrementalLogOpts(ctx context.Context, mirrorPath string, priorTips map[string]string) string {
	if len(priorTips) == 0 {
		return "" // no baseline yet: caller should do a full scan
	}

	args := []string{"--full-history", "--all", "--diff-filter=tuxdb"}
	seen := map[string]bool{}
	for _, sha := range priorTips {
		if seen[sha] || !ObjectExists(ctx, mirrorPath, sha) {
			continue
		}
		seen[sha] = true
		args = append(args, "^"+sha)
	}
	return strings.Join(args, " ")
}

func run(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return nil
}

func output(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String(), nil
}

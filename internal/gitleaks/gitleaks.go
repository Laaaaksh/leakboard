// Package gitleaks shells out to the gitleaks CLI to scan a git repository
// and parses its JSON report. Leakboard does not reimplement secret
// detection: gitleaks' rule set is the engine, this package is the thin
// adapter that runs it and turns its report into Go values.
package gitleaks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Finding mirrors the subset of gitleaks' JSON report fields Leakboard
// stores and displays.
type Finding struct {
	RuleID      string    `json:"RuleID"`
	Description string    `json:"Description"`
	StartLine   int       `json:"StartLine"`
	EndLine     int       `json:"EndLine"`
	Match       string    `json:"Match"`
	Secret      string    `json:"Secret"`
	File        string    `json:"File"`
	Commit      string    `json:"Commit"`
	Author      string    `json:"Author"`
	Email       string    `json:"Email"`
	Date        time.Time `json:"Date"`
	Fingerprint string    `json:"Fingerprint"`
}

// Scanner runs the gitleaks binary at Path.
type Scanner struct {
	Path string
}

// New returns a Scanner that runs the gitleaks binary at path ("gitleaks" if empty).
func New(path string) *Scanner {
	if path == "" {
		path = "gitleaks"
	}
	return &Scanner{Path: path}
}

// CheckAvailable verifies the configured gitleaks binary can be executed,
// so startup fails loudly instead of every scan silently erroring later.
func (s *Scanner) CheckAvailable(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, s.Path, "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gitleaks binary %q not usable (set LEAKBOARD_GITLEAKS_PATH): %w", s.Path, err)
	}
	return nil
}

// Scan runs `gitleaks git` against the repository at repoPath (a working
// tree or a bare/mirror clone both work). An empty logOpts scans full
// history on every reachable ref; a non-empty logOpts (e.g.
// "--all ^<oldsha1> ^<oldsha2>") restricts the scan to commits introduced
// since a prior baseline.
func (s *Scanner) Scan(ctx context.Context, repoPath string, logOpts string) ([]Finding, error) {
	reportFile, err := os.CreateTemp("", "leakboard-report-*.json")
	if err != nil {
		return nil, fmt.Errorf("create report temp file: %w", err)
	}
	reportPath := reportFile.Name()
	_ = reportFile.Close()
	defer func() { _ = os.Remove(reportPath) }()

	args := []string{"git", repoPath, "--report-format", "json", "--report-path", reportPath,
		"--exit-code", "0", "--no-banner", "--log-level", "error"}
	if logOpts != "" {
		args = append(args, "--log-opts", logOpts)
	}

	cmd := exec.CommandContext(ctx, s.Path, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gitleaks git failed: %w: %s", err, stderr.String())
	}

	data, err := os.ReadFile(reportPath)
	if err != nil {
		return nil, fmt.Errorf("read gitleaks report: %w", err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	var findings []Finding
	if err := json.Unmarshal(data, &findings); err != nil {
		return nil, fmt.Errorf("parse gitleaks report: %w", err)
	}
	return findings, nil
}

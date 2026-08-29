# Changelog

All notable changes to Leakboard are documented in this file. Released sections
mirror the notes on the [GitHub Releases page](https://github.com/Laaaaksh/leakboard/releases),
condensed into user-facing terms. Format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Scheduled gitleaks scanning of connected GitHub organizations and manually
  added repos, with an all-branch incremental scan after the first baseline.
- A findings dashboard with per-finding status (new / acknowledged / resolved
  / false positive), deduplicated by gitleaks' own fingerprint.
- New-finding alerting to Slack, Discord, or a generic JSON webhook — fired
  once per genuinely new finding, never on a rescan of an already-known one.
- An org-wide allowlist (by rule ID, path glob, or regex) and per-finding
  "mark as false positive," both of which suppress future findings.
- A single Go binary with an embedded dashboard frontend, backed by Postgres.

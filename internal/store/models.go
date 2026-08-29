package store

import "time"

// FindingStatus is the lifecycle state of a detected secret.
type FindingStatus string

// Finding lifecycle states.
const (
	StatusNew           FindingStatus = "new"
	StatusAcknowledged  FindingStatus = "acknowledged"
	StatusResolved      FindingStatus = "resolved"
	StatusFalsePositive FindingStatus = "false_positive"
)

// User is a dashboard login account. Leakboard has exactly one per instance.
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// Connection is a connected GitHub organization: its repos are listed and
// scanned using AccessToken.
type Connection struct {
	ID          int64
	Name        string
	GitHubOrg   string
	AccessToken string
	CreatedAt   time.Time
}

// Repo is a single git repository tracked for scanning.
type Repo struct {
	ID               int64
	ConnectionID     *int64
	Name             string
	CloneURL         string
	DefaultBranch    string
	MirrorPath       string
	ScannedRefTips   map[string]string
	LastScannedAt    *time.Time
	LastScanError    string
	ScanIntervalSecs int
	CreatedAt        time.Time
}

// ScanRun records one scan attempt of a repo, for the recent-activity view.
type ScanRun struct {
	ID          int64
	RepoID      int64
	StartedAt   time.Time
	FinishedAt  *time.Time
	Status      string
	NewFindings int
	Error       string
}

// Finding is one detected secret, deduplicated per-repo by Fingerprint
// (gitleaks' own `commit:file:rule:line` identity).
type Finding struct {
	ID           int64
	RepoID       int64
	RepoName     string
	Fingerprint  string
	RuleID       string
	Description  string
	FilePath     string
	StartLine    int
	EndLine      int
	CommitSHA    string
	CommitAuthor string
	CommitEmail  string
	CommitDate   *time.Time
	Secret       string
	Match        string
	Status       FindingStatus
	FirstSeenAt  time.Time
	LastSeenAt   time.Time
	ResolvedAt   *time.Time
}

// AllowlistEntry suppresses future findings. Exactly one of RuleID+PathPattern,
// Regex, or Fingerprint should be set: fingerprint entries suppress one exact
// finding ("mark as false positive" from the UI); rule/path or regex entries
// suppress a whole class, mirroring gitleaks' own [allowlist] config shape.
type AllowlistEntry struct {
	ID          int64
	RuleID      string
	PathPattern string
	Regex       string
	Fingerprint string
	Reason      string
	CreatedAt   time.Time
}

// WebhookKind selects how an alert notification is formatted.
type WebhookKind string

// Supported webhook kinds.
const (
	WebhookSlack   WebhookKind = "slack"
	WebhookDiscord WebhookKind = "discord"
	WebhookGeneric WebhookKind = "generic"
)

// Webhook is an outbound alert destination fired on new findings.
type Webhook struct {
	ID        int64
	Kind      WebhookKind
	TargetURL string
	Enabled   bool
	CreatedAt time.Time
}

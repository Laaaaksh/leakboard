package scanner

import (
	"testing"

	"github.com/Laaaaksh/leakboard/internal/gitleaks"
	"github.com/Laaaaksh/leakboard/internal/store"
)

func TestFilterAllowlisted(t *testing.T) {
	findings := []gitleaks.Finding{
		{RuleID: "aws-access-token", File: "src/main.go", Secret: "AKIAABCDEFGHIJKLMNOP"},
		{RuleID: "generic-api-key", File: "testdata/fixtures/sample.env", Secret: "sk-fake-1234"},
		{RuleID: "generic-api-key", File: "src/config.go", Secret: "sk-live-realsecret999"},
	}

	tests := []struct {
		name    string
		entries []store.AllowlistEntry
		want    []string // expected remaining Secret values, in order
	}{
		{
			name:    "no entries keeps everything",
			entries: nil,
			want:    []string{"AKIAABCDEFGHIJKLMNOP", "sk-fake-1234", "sk-live-realsecret999"},
		},
		{
			name:    "path glob suppresses only matching path",
			entries: []store.AllowlistEntry{{PathPattern: "testdata/fixtures/*"}},
			want:    []string{"AKIAABCDEFGHIJKLMNOP", "sk-live-realsecret999"},
		},
		{
			name:    "rule id alone suppresses every finding for that rule",
			entries: []store.AllowlistEntry{{RuleID: "generic-api-key"}},
			want:    []string{"AKIAABCDEFGHIJKLMNOP"},
		},
		{
			name:    "rule id plus path only suppresses the intersection",
			entries: []store.AllowlistEntry{{RuleID: "generic-api-key", PathPattern: "testdata/fixtures/*"}},
			want:    []string{"AKIAABCDEFGHIJKLMNOP", "sk-live-realsecret999"},
		},
		{
			name:    "regex matches secret value",
			entries: []store.AllowlistEntry{{Regex: `^sk-fake-`}},
			want:    []string{"AKIAABCDEFGHIJKLMNOP", "sk-live-realsecret999"},
		},
		{
			name:    "invalid regex is ignored, not fatal",
			entries: []store.AllowlistEntry{{Regex: `(unterminated`}},
			want:    []string{"AKIAABCDEFGHIJKLMNOP", "sk-fake-1234", "sk-live-realsecret999"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FilterAllowlisted(findings, tc.entries)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d findings, want %d: %+v", len(got), len(tc.want), got)
			}
			for i, f := range got {
				if f.Secret != tc.want[i] {
					t.Errorf("finding %d: got secret %q, want %q", i, f.Secret, tc.want[i])
				}
			}
		})
	}
}

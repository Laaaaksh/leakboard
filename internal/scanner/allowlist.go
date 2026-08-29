package scanner

import (
	"path"
	"regexp"

	"github.com/Laaaaksh/leakboard/internal/gitleaks"
	"github.com/Laaaaksh/leakboard/internal/store"
)

// FilterAllowlisted drops findings matched by a rule+path or regex allowlist
// entry, so a known-safe pattern (a test fixture, a documented placeholder)
// never even reaches the findings table for any repo in the org.
// Fingerprint-only entries (single "mark as false positive" clicks) are
// intentionally not applied here: they target one exact finding already in
// the database, not a whole class of future ones.
func FilterAllowlisted(findings []gitleaks.Finding, entries []store.AllowlistEntry) []gitleaks.Finding {
	var rulePath []store.AllowlistEntry
	var regexes []*regexp.Regexp
	for _, e := range entries {
		switch {
		case e.Regex != "":
			if re, err := regexp.Compile(e.Regex); err == nil {
				regexes = append(regexes, re)
			}
		case e.RuleID != "" || e.PathPattern != "":
			rulePath = append(rulePath, e)
		}
	}

	if len(rulePath) == 0 && len(regexes) == 0 {
		return findings
	}

	out := findings[:0:0]
	for _, f := range findings {
		if isAllowlisted(f, rulePath, regexes) {
			continue
		}
		out = append(out, f)
	}
	return out
}

func isAllowlisted(f gitleaks.Finding, rulePath []store.AllowlistEntry, regexes []*regexp.Regexp) bool {
	for _, e := range rulePath {
		ruleMatches := e.RuleID == "" || e.RuleID == f.RuleID
		pathMatches := e.PathPattern == "" || globMatch(e.PathPattern, f.File)
		if ruleMatches && pathMatches {
			return true
		}
	}
	for _, re := range regexes {
		if re.MatchString(f.Secret) || re.MatchString(f.Match) {
			return true
		}
	}
	return false
}

func globMatch(pattern, name string) bool {
	ok, err := path.Match(pattern, name)
	return err == nil && ok
}

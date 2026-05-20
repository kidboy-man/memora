package domain

import "regexp"

// secretPatterns contains high-confidence regexes for common credential formats.
// False positives (blocking valid content) are worse than false negatives here,
// so only well-defined, high-entropy patterns are included.
var secretPatterns = []*regexp.Regexp{
	// AWS access key ID: always starts with AKIA, followed by 16 uppercase alphanumeric chars
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	// GitHub personal access token
	regexp.MustCompile(`ghp_[A-Za-z0-9]{36}`),
	// GitHub app installation token
	regexp.MustCompile(`ghs_[A-Za-z0-9]{36}`),
	// OpenAI-style API key (also used by OpenRouter, Anthropic, etc.)
	regexp.MustCompile(`sk-[A-Za-z0-9]{32,}`),
	// Generic API key assignment pattern: apikey="<value>" / api_key: <value>
	regexp.MustCompile(`(?i)api[-_]?key["'\s]*[:=]["'\s]*[A-Za-z0-9\-_]{20,}`),
}

// ScanForSecrets returns true if content likely contains a credential.
//
// This is a best-effort scan — not exhaustive. False negatives are possible.
// A positive result sets SecretWarning=true in the RememberResult but does NOT
// block the write. The agent decides whether to redact and retry.
func ScanForSecrets(content string) bool {
	for _, p := range secretPatterns {
		if p.MatchString(content) {
			return true
		}
	}
	return false
}

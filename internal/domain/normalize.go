package domain

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

// NormalizeContent prepares content for hashing and dedup comparison.
//
// Order matters:
//  1. Unicode NFC — compose characters before lowercasing. Some codepoints
//     change form after ToLower, so NFC must run first to guarantee that
//     "café" (precomposed U+00E9) and "café" (e + U+0301 combining accent)
//     always produce the same hash.
//  2. Lowercase — case-insensitive dedup.
//  3. TrimSpace — strip leading/trailing whitespace.
//  4. Collapse internal runs — "foo  bar" → "foo bar".
func NormalizeContent(content string) string {
	s := norm.NFC.String(content)
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}

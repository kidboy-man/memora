package domain_test

import (
	"testing"

	"github.com/kidboy-man/memora/internal/domain"
)

func TestNormalizeContent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "already normalized",
			input: "the sky is blue",
			want:  "the sky is blue",
		},
		{
			name:  "leading and trailing whitespace",
			input: "  hello world  ",
			want:  "hello world",
		},
		{
			name:  "internal whitespace collapse",
			input: "foo  bar\t\tbaz",
			want:  "foo bar baz",
		},
		{
			name:  "uppercase to lowercase",
			input: "The Sky Is BLUE",
			want:  "the sky is blue",
		},
		{
			name:  "mixed case and whitespace",
			input: "  Hello   WORLD  ",
			want:  "hello world",
		},
		{
			name:  "NFC precomposed equals decomposed",
			input: "café", // é as single codepoint U+00E9
			want:  "café",
		},
		{
			name:  "NFC decomposed normalizes to precomposed",
			input: "café", // é as e + combining accent U+0301
			want:  "café",  // NFC composes to U+00E9
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "whitespace only",
			input: "   \t\n  ",
			want:  "",
		},
		{
			name:  "newlines collapsed",
			input: "line one\nline two",
			want:  "line one line two",
		},
		{
			name:  "tabs collapsed",
			input: "col1\tcol2\tcol3",
			want:  "col1 col2 col3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.NormalizeContent(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeContent(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestNormalizeContentIdempotent verifies that normalizing an already-normalized
// string produces the same result (idempotency is required for re-hashing safety).
func TestNormalizeContentIdempotent(t *testing.T) {
	inputs := []string{
		"the sky is blue",
		"café au lait",
		"hello world",
		"",
	}
	for _, input := range inputs {
		once := domain.NormalizeContent(input)
		twice := domain.NormalizeContent(once)
		if once != twice {
			t.Errorf("NormalizeContent not idempotent for %q: first=%q second=%q", input, once, twice)
		}
	}
}

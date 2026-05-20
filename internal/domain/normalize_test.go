package domain_test

import (
	"testing"

	"github.com/kidboy-man/memora/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeContent(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "already normalized", input: "the sky is blue", want: "the sky is blue"},
		{name: "leading and trailing whitespace", input: "  hello world  ", want: "hello world"},
		{name: "internal whitespace collapse", input: "foo  bar\t\tbaz", want: "foo bar baz"},
		{name: "uppercase to lowercase", input: "The Sky Is BLUE", want: "the sky is blue"},
		{name: "mixed case and whitespace", input: "  Hello   WORLD  ", want: "hello world"},
		{name: "NFC precomposed equals decomposed", input: "café", want: "café"},        // U+00E9 precomposed
		{name: "NFC decomposed normalizes to precomposed", input: "café", want: "café"}, // e + U+0301
		{name: "empty string", input: "", want: ""},
		{name: "whitespace only", input: "   \t\n  ", want: ""},
		{name: "newlines collapsed", input: "line one\nline two", want: "line one line two"},
		{name: "tabs collapsed", input: "col1\tcol2\tcol3", want: "col1 col2 col3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, domain.NormalizeContent(tt.input))
		})
	}
}

// TestNormalizeContentIdempotent verifies that normalizing an already-normalized
// string produces the same result (required for re-hashing safety).
func TestNormalizeContentIdempotent(t *testing.T) {
	inputs := []string{"the sky is blue", "café au lait", "hello world", ""}
	for _, input := range inputs {
		once := domain.NormalizeContent(input)
		twice := domain.NormalizeContent(once)
		assert.Equal(t, once, twice, "NormalizeContent not idempotent for %q", input)
	}
}

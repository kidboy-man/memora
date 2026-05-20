package domain_test

import (
	"testing"

	"github.com/kidboy-man/memora/internal/domain"
)

func TestScanForSecrets(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "clean content",
			content: "the sky is blue and the grass is green",
			want:    false,
		},
		{
			name:    "empty string",
			content: "",
			want:    false,
		},
		{
			name:    "AWS access key",
			content: "use key AKIAIOSFODNN7EXAMPLE for boto3",
			want:    true,
		},
		{
			name:    "GitHub personal access token ghp_",
			content: "token: ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij",
			want:    true,
		},
		{
			name:    "GitHub app installation token ghs_",
			content: "ghs_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij is the token",
			want:    true,
		},
		{
			name:    "OpenAI-style sk- key",
			content: "OPENAI_API_KEY=sk-ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij",
			want:    true,
		},
		{
			name:    "generic api_key assignment",
			content: `api_key = "supersecretvalue1234567890123456"`,
			want:    true,
		},
		{
			name:    "generic apikey colon assignment",
			content: "apikey: averylongsecretkeyvalue1234567890",
			want:    true,
		},
		{
			name:    "aws key embedded in sentence",
			content: "I was using AKIAIOSFODNN7EXAMPLE to access S3 buckets",
			want:    true,
		},
		{
			name:    "short sk- prefix not matched (< 32 chars after prefix)",
			content: "sk-short",
			want:    false,
		},
		{
			name:    "normal text mentioning api key concept without value",
			content: "you need to provide an api key to authenticate",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.ScanForSecrets(tt.content)
			if got != tt.want {
				t.Errorf("ScanForSecrets(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

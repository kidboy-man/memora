package domain_test

import (
	"testing"

	"github.com/kidboy-man/memora/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddingProfileValidate(t *testing.T) {
	validBase := func() domain.EmbeddingProfile {
		return domain.EmbeddingProfile{
			Name:           "openai-text-3-small",
			Provider:       domain.ProviderOpenAI,
			Model:          "text-embedding-3-small",
			Dimensions:     1536,
			DistanceMetric: domain.DistanceCosine,
		}
	}

	tests := []struct {
		name        string
		mutate      func(*domain.EmbeddingProfile)
		wantErr     bool
		errContains []string
	}{
		{
			name:    "valid openai profile",
			mutate:  func(_ *domain.EmbeddingProfile) {},
			wantErr: false,
		},
		{
			name: "valid ollama profile with empty api base url",
			mutate: func(p *domain.EmbeddingProfile) {
				p.Provider = domain.ProviderOllama
				p.Model = "nomic-embed-text"
				p.Dimensions = 768
				p.APIBaseURL = ""
			},
			wantErr: false,
		},
		{
			name: "valid openrouter profile with custom base url",
			mutate: func(p *domain.EmbeddingProfile) {
				p.Provider = domain.ProviderOpenRouter
				p.Model = "openai/text-embedding-3-small"
				p.APIBaseURL = "https://openrouter.ai/api/v1"
			},
			wantErr: false,
		},
		{
			name: "valid l2 distance metric",
			mutate: func(p *domain.EmbeddingProfile) {
				p.DistanceMetric = domain.DistanceL2
			},
			wantErr: false,
		},
		{
			name: "valid ip distance metric",
			mutate: func(p *domain.EmbeddingProfile) {
				p.DistanceMetric = domain.DistanceIP
			},
			wantErr: false,
		},
		{
			name: "empty name",
			mutate: func(p *domain.EmbeddingProfile) {
				p.Name = ""
			},
			wantErr:     true,
			errContains: []string{"name is required"},
		},
		{
			name: "invalid provider",
			mutate: func(p *domain.EmbeddingProfile) {
				p.Provider = "huggingface"
			},
			wantErr:     true,
			errContains: []string{"invalid provider"},
		},
		{
			name: "empty provider",
			mutate: func(p *domain.EmbeddingProfile) {
				p.Provider = ""
			},
			wantErr:     true,
			errContains: []string{"invalid provider"},
		},
		{
			name: "empty model",
			mutate: func(p *domain.EmbeddingProfile) {
				p.Model = ""
			},
			wantErr:     true,
			errContains: []string{"model is required"},
		},
		{
			name: "zero dimensions",
			mutate: func(p *domain.EmbeddingProfile) {
				p.Dimensions = 0
			},
			wantErr:     true,
			errContains: []string{"dimensions"},
		},
		{
			name: "negative dimensions",
			mutate: func(p *domain.EmbeddingProfile) {
				p.Dimensions = -1
			},
			wantErr:     true,
			errContains: []string{"dimensions"},
		},
		{
			name: "invalid distance metric",
			mutate: func(p *domain.EmbeddingProfile) {
				p.DistanceMetric = "euclidean"
			},
			wantErr:     true,
			errContains: []string{"invalid distance metric"},
		},
		{
			name: "empty distance metric",
			mutate: func(p *domain.EmbeddingProfile) {
				p.DistanceMetric = ""
			},
			wantErr:     true,
			errContains: []string{"invalid distance metric"},
		},
		{
			name: "multiple errors collected",
			mutate: func(p *domain.EmbeddingProfile) {
				p.Name = ""
				p.Provider = "bad"
				p.Model = ""
				p.Dimensions = 0
				p.DistanceMetric = ""
			},
			wantErr:     true,
			errContains: []string{"name", "provider", "model", "dimensions", "distance metric"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validBase()
			tt.mutate(&p)

			err := p.Validate()

			if tt.wantErr {
				require.Error(t, err)
				var ve *domain.ValidationError
				require.ErrorAs(t, err, &ve)
				for _, sub := range tt.errContains {
					assert.Contains(t, err.Error(), sub)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

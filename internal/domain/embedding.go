package domain

import (
	"fmt"
	"strings"
	"time"
)

// DistanceMetric selects the pgvector operator used for similarity search.
type DistanceMetric string

const (
	DistanceCosine DistanceMetric = "cosine"
	DistanceL2     DistanceMetric = "l2"
	DistanceIP     DistanceMetric = "ip"
)

var validDistanceMetrics = map[DistanceMetric]bool{
	DistanceCosine: true,
	DistanceL2:     true,
	DistanceIP:     true,
}

// Provider constants for embedding adapters. Defined in the domain so the
// adapter factory (internal/adapter/embedding) can reference these without
// introducing a circular dependency. Adding a new provider requires updating
// both this set and implementing a new adapter.
const (
	ProviderOpenRouter = "openrouter"
	ProviderOpenAI     = "openai"
	ProviderOllama     = "ollama"
)

var validProviders = map[string]bool{
	ProviderOpenRouter: true,
	ProviderOpenAI:     true,
	ProviderOllama:     true,
}

// EmbeddingProfile holds the configuration for one embedding provider/model
// combination. Exactly one profile may be active at a time; the active profile
// is used by RememberService and UpdateService to generate vectors.
type EmbeddingProfile struct {
	ID             string
	Name           string
	Provider       string
	Model          string
	APIBaseURL     string // empty = adapter uses its built-in default
	Dimensions     int
	DistanceMetric DistanceMetric
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Validate checks all domain invariants for an EmbeddingProfile.
func (p *EmbeddingProfile) Validate() error {
	var errs []string

	if strings.TrimSpace(p.Name) == "" {
		errs = append(errs, "name is required")
	}
	if !validProviders[p.Provider] {
		errs = append(errs, fmt.Sprintf("invalid provider %q; valid: openrouter, openai, ollama", p.Provider))
	}
	if strings.TrimSpace(p.Model) == "" {
		errs = append(errs, "model is required")
	}
	if p.Dimensions <= 0 {
		errs = append(errs, fmt.Sprintf("dimensions %d must be > 0", p.Dimensions))
	}
	if !validDistanceMetrics[p.DistanceMetric] {
		errs = append(errs, fmt.Sprintf("invalid distance metric %q; valid: cosine, l2, ip", p.DistanceMetric))
	}

	if len(errs) > 0 {
		return &ValidationError{Fields: errs}
	}
	return nil
}

// MemoryEmbedding stores a vector representation of one memory under one
// embedding profile. Stored separately from Memory so canonical content
// survives provider changes and re-embedding can be done safely.
type MemoryEmbedding struct {
	MemoryID           string
	ProfileID          string
	Embedding          []float32 // raw vector; pgx handles BYTEA ↔ []float32 via pgvector-go
	EmbeddingDimension int
	ContentHash        []byte // mirrors memory.ContentHash; skip re-embed when hash unchanged
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

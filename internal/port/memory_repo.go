package port

import (
	"context"

	"github.com/kidboy-man/memora/internal/domain"
)

type MemoryFilter struct {
	Scope          domain.Scope
	Project        string
	IncludeGlobal  bool
	Types          []domain.MemoryType
	Tags           []string
	Source         string
	IncludeDeleted bool
	Cursor         string
	Limit          int
}

type MemoryUpdates struct {
	Content     *string
	ContentHash []byte
	Tags        *[]string
	Metadata    *map[string]any
	Confidence  *float64
}

type SimilarResult struct {
	MemoryID string
	Score    float64
}

type MemoryRepository interface {
	InsertMemory(ctx context.Context, tx Tx, memory *domain.Memory) (string, error)
	GetMemoryByID(ctx context.Context, id string, includeDeleted bool) (*domain.Memory, error)
	ExistsExact(ctx context.Context, scope domain.Scope, project string, memType domain.MemoryType, contentHash []byte) (bool, string, error)
	FindSimilar(ctx context.Context, profileID string, embedding []float32, threshold float64, scope domain.Scope, project string, memType domain.MemoryType, limit int) ([]SimilarResult, error)
	UpdateMemory(ctx context.Context, tx Tx, id string, expectedVersion int, updates MemoryUpdates) (*domain.Memory, error)
	SoftDeleteMemory(ctx context.Context, tx Tx, id string, deletedBy string, reason string) error
	SearchSemantic(ctx context.Context, profileID string, embedding []float32, filter MemoryFilter) ([]*domain.Memory, []float64, error)
	SearchKeyword(ctx context.Context, query string, filter MemoryFilter) ([]*domain.Memory, []float64, error)
	ListMemories(ctx context.Context, filter MemoryFilter) ([]*domain.Memory, string, int, error)
	CountMemories(ctx context.Context) (int, int, map[string]int, error)
}

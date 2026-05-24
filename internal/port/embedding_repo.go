package port

import (
	"context"

	"github.com/kidboy-man/memora/internal/domain"
)

type EmbeddingProfileRepository interface {
	InsertProfile(ctx context.Context, tx Tx, profile *domain.EmbeddingProfile) (string, error)
	GetProfileByID(ctx context.Context, id string) (*domain.EmbeddingProfile, error)
	GetActiveProfile(ctx context.Context) (*domain.EmbeddingProfile, error)
	ListProfiles(ctx context.Context) ([]*domain.EmbeddingProfile, error)
	ActivateProfile(ctx context.Context, tx Tx, id string) error
}

type MemoryEmbeddingRepository interface {
	UpsertEmbedding(ctx context.Context, tx Tx, embedding *domain.MemoryEmbedding) error
	GetEmbedding(ctx context.Context, memoryID string, profileID string) (*domain.MemoryEmbedding, error)
}

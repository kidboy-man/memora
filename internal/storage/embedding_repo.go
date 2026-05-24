package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kidboy-man/memora/internal/domain"
	"github.com/kidboy-man/memora/internal/port"
	"github.com/pgvector/pgvector-go"
)

type EmbeddingProfileRepository struct {
	pool *pgxpool.Pool
}

func NewEmbeddingProfileRepository(pool *pgxpool.Pool) *EmbeddingProfileRepository {
	return &EmbeddingProfileRepository{pool: pool}
}

func (r *EmbeddingProfileRepository) InsertProfile(ctx context.Context, tx port.Tx, profile *domain.EmbeddingProfile) (string, error) {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return "", err
	}

	apiBaseURL := any(nil)
	if profile.APIBaseURL != "" {
		apiBaseURL = profile.APIBaseURL
	}

	var id string
	err = pgxTx.QueryRow(ctx, `
		INSERT INTO embedding_profiles (name, provider, model, api_base_url, dimensions, distance_metric, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, profile.Name, profile.Provider, profile.Model, apiBaseURL, profile.Dimensions, profile.DistanceMetric, profile.IsActive).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert embedding profile: %w", err)
	}
	return id, nil
}

func (r *EmbeddingProfileRepository) GetProfileByID(ctx context.Context, id string) (*domain.EmbeddingProfile, error) {
	profile, err := scanEmbeddingProfile(r.pool.QueryRow(ctx, `
		SELECT id, name, provider, model, coalesce(api_base_url, ''), dimensions, distance_metric, is_active, created_at, updated_at
		FROM embedding_profiles
		WHERE id = $1
	`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: "embedding profile", ID: id}
		}
		return nil, fmt.Errorf("get embedding profile: %w", err)
	}
	return profile, nil
}

func (r *EmbeddingProfileRepository) GetActiveProfile(ctx context.Context) (*domain.EmbeddingProfile, error) {
	profile, err := scanEmbeddingProfile(r.pool.QueryRow(ctx, `
		SELECT id, name, provider, model, coalesce(api_base_url, ''), dimensions, distance_metric, is_active, created_at, updated_at
		FROM embedding_profiles
		WHERE is_active = TRUE
	`))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &domain.NoActiveProfileError{}
		}
		return nil, fmt.Errorf("get active embedding profile: %w", err)
	}
	return profile, nil
}

func (r *EmbeddingProfileRepository) ListProfiles(ctx context.Context) ([]*domain.EmbeddingProfile, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, provider, model, coalesce(api_base_url, ''), dimensions, distance_metric, is_active, created_at, updated_at
		FROM embedding_profiles
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list embedding profiles: %w", err)
	}
	defer rows.Close()

	var profiles []*domain.EmbeddingProfile
	for rows.Next() {
		profile, err := scanEmbeddingProfile(rows)
		if err != nil {
			return nil, fmt.Errorf("scan embedding profile: %w", err)
		}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate embedding profiles: %w", err)
	}
	return profiles, nil
}

func (r *EmbeddingProfileRepository) ActivateProfile(ctx context.Context, tx port.Tx, id string) error {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return err
	}

	if _, err := pgxTx.Exec(ctx, `UPDATE embedding_profiles SET is_active = FALSE, updated_at = NOW() WHERE is_active = TRUE`); err != nil {
		return fmt.Errorf("deactivate embedding profiles: %w", err)
	}
	commandTag, err := pgxTx.Exec(ctx, `UPDATE embedding_profiles SET is_active = TRUE, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("activate embedding profile: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return &domain.NotFoundError{Resource: "embedding profile", ID: id}
	}
	return nil
}

type MemoryEmbeddingRepository struct {
	pool *pgxpool.Pool
}

func NewMemoryEmbeddingRepository(pool *pgxpool.Pool) *MemoryEmbeddingRepository {
	return &MemoryEmbeddingRepository{pool: pool}
}

func (r *MemoryEmbeddingRepository) UpsertEmbedding(ctx context.Context, tx port.Tx, embedding *domain.MemoryEmbedding) error {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return err
	}

	_, err = pgxTx.Exec(ctx, `
		INSERT INTO memory_embeddings (memory_id, profile_id, embedding, embedding_dimension, content_hash)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (memory_id, profile_id) DO UPDATE
		SET embedding = EXCLUDED.embedding,
		    embedding_dimension = EXCLUDED.embedding_dimension,
		    content_hash = EXCLUDED.content_hash,
		    updated_at = NOW()
	`, embedding.MemoryID, embedding.ProfileID, pgvector.NewVector(embedding.Embedding), embedding.EmbeddingDimension, embedding.ContentHash)
	if err != nil {
		return fmt.Errorf("upsert memory embedding: %w", err)
	}
	return nil
}

func (r *MemoryEmbeddingRepository) GetEmbedding(ctx context.Context, memoryID string, profileID string) (*domain.MemoryEmbedding, error) {
	embedding, err := scanMemoryEmbedding(r.pool.QueryRow(ctx, `
		SELECT memory_id, profile_id, embedding, embedding_dimension, content_hash, created_at, updated_at
		FROM memory_embeddings
		WHERE memory_id = $1 AND profile_id = $2
	`, memoryID, profileID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: "memory embedding", ID: memoryID}
		}
		return nil, fmt.Errorf("get memory embedding: %w", err)
	}
	return embedding, nil
}

func scanEmbeddingProfile(row pgx.Row) (*domain.EmbeddingProfile, error) {
	var profile domain.EmbeddingProfile
	err := row.Scan(
		&profile.ID,
		&profile.Name,
		&profile.Provider,
		&profile.Model,
		&profile.APIBaseURL,
		&profile.Dimensions,
		&profile.DistanceMetric,
		&profile.IsActive,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func scanMemoryEmbedding(row pgx.Row) (*domain.MemoryEmbedding, error) {
	var embedding domain.MemoryEmbedding
	var vector pgvector.Vector
	err := row.Scan(
		&embedding.MemoryID,
		&embedding.ProfileID,
		&vector,
		&embedding.EmbeddingDimension,
		&embedding.ContentHash,
		&embedding.CreatedAt,
		&embedding.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	embedding.Embedding = vector.Slice()
	return &embedding, nil
}

var _ port.EmbeddingProfileRepository = (*EmbeddingProfileRepository)(nil)
var _ port.MemoryEmbeddingRepository = (*MemoryEmbeddingRepository)(nil)

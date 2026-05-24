//go:build integration

package storage

import (
	"context"
	"testing"
	"time"

	"github.com/kidboy-man/memora/internal/domain"
	"github.com/kidboy-man/memora/internal/port"
	"github.com/stretchr/testify/require"
)

func TestEmbeddingRepositoriesProfilesAndVectors(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	dsn := startPostgres(t, ctx)
	require.NoError(t, RunMigrations(ctx, dsn, DirectionUp))

	dbCfg, err := configFromDSN(dsn)
	require.NoError(t, err)

	pool, err := NewPool(ctx, dbCfg)
	require.NoError(t, err)
	defer pool.Close()

	runner := NewTxRunner(pool)
	profiles := NewEmbeddingProfileRepository(pool)
	embeddings := NewMemoryEmbeddingRepository(pool)
	memories := NewMemoryRepository(pool)

	profile := &domain.EmbeddingProfile{
		Name:           "test-profile",
		Provider:       domain.ProviderOpenAI,
		Model:          "text-embedding-3-small",
		Dimensions:     3,
		DistanceMetric: domain.DistanceCosine,
		IsActive:       true,
	}

	require.NoError(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		id, err := profiles.InsertProfile(ctx, tx, profile)
		require.NoError(t, err)
		profile.ID = id
		return nil
	}))

	active, err := profiles.GetActiveProfile(ctx)
	require.NoError(t, err)
	require.Equal(t, profile.ID, active.ID)
	require.True(t, active.IsActive)

	byID, err := profiles.GetProfileByID(ctx, profile.ID)
	require.NoError(t, err)
	require.Equal(t, profile.Name, byID.Name)
	require.Equal(t, profile.DistanceMetric, byID.DistanceMetric)

	listed, err := profiles.ListProfiles(ctx)
	require.NoError(t, err)
	require.Len(t, listed, 1)

	memory := &domain.Memory{
		Content:     "Vector search should return nearby memory.",
		ContentHash: domain.SHA256Hash("vector search should return nearby memory."),
		Type:        domain.TypeFact,
		Scope:       domain.ScopeProject,
		Project:     "memora",
		Source:      "integration-test",
		Tags:        []string{"vector"},
		Metadata:    map[string]any{},
		Confidence:  1,
	}
	require.NoError(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		id, err := memories.InsertMemory(ctx, tx, memory)
		require.NoError(t, err)
		memory.ID = id
		return embeddings.UpsertEmbedding(ctx, tx, &domain.MemoryEmbedding{
			MemoryID:           memory.ID,
			ProfileID:          profile.ID,
			Embedding:          []float32{1, 0, 0},
			EmbeddingDimension: 3,
			ContentHash:        memory.ContentHash,
		})
	}))

	stored, err := embeddings.GetEmbedding(ctx, memory.ID, profile.ID)
	require.NoError(t, err)
	require.Equal(t, []float32{1, 0, 0}, stored.Embedding)
	require.Equal(t, 3, stored.EmbeddingDimension)

	similar, err := memories.FindSimilar(ctx, profile.ID, []float32{1, 0, 0}, 0.99, domain.ScopeProject, "memora", domain.TypeFact, 10)
	require.NoError(t, err)
	require.Len(t, similar, 1)
	require.Equal(t, memory.ID, similar[0].MemoryID)
	require.InDelta(t, 1.0, similar[0].Score, 0.001)

	semantic, scores, err := memories.SearchSemantic(ctx, profile.ID, []float32{1, 0, 0}, port.MemoryFilter{Scope: domain.ScopeProject, Project: "memora", Limit: 10})
	require.NoError(t, err)
	require.Len(t, semantic, 1)
	require.Len(t, scores, 1)
	require.Equal(t, memory.ID, semantic[0].ID)
	require.InDelta(t, 1.0, scores[0], 0.001)

	globalMemory := &domain.Memory{
		Content:     "Global vector memory should join semantic project results.",
		ContentHash: domain.SHA256Hash("global vector memory should join semantic project results."),
		Type:        domain.TypeDecision,
		Scope:       domain.ScopeGlobal,
		Source:      "integration-test",
		Tags:        []string{"vector"},
		Metadata:    map[string]any{},
		Confidence:  1,
	}
	require.NoError(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		id, err := memories.InsertMemory(ctx, tx, globalMemory)
		require.NoError(t, err)
		globalMemory.ID = id
		return embeddings.UpsertEmbedding(ctx, tx, &domain.MemoryEmbedding{
			MemoryID:           globalMemory.ID,
			ProfileID:          profile.ID,
			Embedding:          []float32{1, 0, 0},
			EmbeddingDimension: 3,
			ContentHash:        globalMemory.ContentHash,
		})
	}))

	semantic, _, err = memories.SearchSemantic(ctx, profile.ID, []float32{1, 0, 0}, port.MemoryFilter{Scope: domain.ScopeProject, Project: "memora", IncludeGlobal: true, Limit: 10})
	require.NoError(t, err)
	require.Len(t, semantic, 2)
	require.ElementsMatch(t, []string{memory.ID, globalMemory.ID}, []string{semantic[0].ID, semantic[1].ID})
}

//go:build integration

package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kidboy-man/memora/internal/domain"
	"github.com/kidboy-man/memora/internal/port"
	"github.com/stretchr/testify/require"
)

func TestMemoryRepositoryInsertGetExactUpdateDelete(t *testing.T) {
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
	repo := NewMemoryRepository(pool)

	memory := &domain.Memory{
		Content:     "Remember that storage adapters use pgx.",
		ContentHash: domain.SHA256Hash("remember that storage adapters use pgx."),
		Type:        domain.TypeFact,
		Scope:       domain.ScopeProject,
		Project:     "memora",
		Source:      "integration-test",
		Tags:        []string{"storage", "pgx"},
		Metadata:    map[string]any{"ticket": "5"},
		Confidence:  0.9,
	}

	require.NoError(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		id, err := repo.InsertMemory(ctx, tx, memory)
		require.NoError(t, err)
		require.NotEmpty(t, id)
		memory.ID = id
		return nil
	}))

	found, err := repo.GetMemoryByID(ctx, memory.ID, false)
	require.NoError(t, err)
	require.Equal(t, memory.ID, found.ID)
	require.Equal(t, memory.Content, found.Content)
	require.Equal(t, memory.ContentHash, found.ContentHash)
	require.Equal(t, memory.Type, found.Type)
	require.Equal(t, memory.Scope, found.Scope)
	require.Equal(t, memory.Project, found.Project)
	require.Equal(t, memory.Source, found.Source)
	require.Equal(t, memory.Tags, found.Tags)
	require.Equal(t, "5", found.Metadata["ticket"])
	require.InDelta(t, 0.9, found.Confidence, 0.001)
	require.Equal(t, 1, found.Version)
	require.Nil(t, found.DeletedAt)

	exists, existingID, err := repo.ExistsExact(ctx, domain.ScopeProject, "memora", domain.TypeFact, memory.ContentHash)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, memory.ID, existingID)

	duplicate := *memory
	duplicate.ID = ""
	require.ErrorAs(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		_, err := repo.InsertMemory(ctx, tx, &duplicate)
		return err
	}), new(*domain.ExactDuplicateError))

	updatedContent := "Remember that storage adapters use pgx transactions."
	updatedTags := []string{"storage", "pgx", "transaction"}
	updatedMetadata := map[string]any{"ticket": "5", "status": "updated"}
	updatedConfidence := 0.8
	var updated *domain.Memory
	require.NoError(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		updated, err = repo.UpdateMemory(ctx, tx, memory.ID, 1, port.MemoryUpdates{
			Content:     &updatedContent,
			ContentHash: domain.SHA256Hash("remember that storage adapters use pgx transactions."),
			Tags:        &updatedTags,
			Metadata:    &updatedMetadata,
			Confidence:  &updatedConfidence,
		})
		return err
	}))
	require.Equal(t, updatedContent, updated.Content)
	require.Equal(t, 2, updated.Version)
	require.Equal(t, updatedTags, updated.Tags)
	require.Equal(t, "updated", updated.Metadata["status"])
	require.InDelta(t, 0.8, updated.Confidence, 0.001)

	var conflictErr *domain.VersionConflictError
	require.ErrorAs(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		_, err := repo.UpdateMemory(ctx, tx, memory.ID, 1, port.MemoryUpdates{Content: &updatedContent})
		return err
	}), &conflictErr)

	require.NoError(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		return repo.SoftDeleteMemory(ctx, tx, memory.ID, "integration-test", "cleanup")
	}))

	_, err = repo.GetMemoryByID(ctx, memory.ID, false)
	var notFound *domain.NotFoundError
	require.ErrorAs(t, err, &notFound)

	deleted, err := repo.GetMemoryByID(ctx, memory.ID, true)
	require.NoError(t, err)
	require.NotNil(t, deleted.DeletedAt)
	require.Equal(t, "integration-test", deleted.DeletedBy)
	require.Equal(t, "cleanup", deleted.DeleteReason)

	var alreadyDeleted *domain.AlreadyDeletedError
	require.ErrorAs(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		return repo.SoftDeleteMemory(ctx, tx, memory.ID, "integration-test", "cleanup")
	}), &alreadyDeleted)

	_, err = repo.GetMemoryByID(ctx, "00000000-0000-0000-0000-000000000000", false)
	require.True(t, errors.As(err, &notFound))
}

func TestMemoryRepositoryListCountAndKeywordSearch(t *testing.T) {
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
	repo := NewMemoryRepository(pool)

	memories := []*domain.Memory{
		{
			Content:     "Global storage decision uses pgxpool.",
			ContentHash: domain.SHA256Hash("global storage decision uses pgxpool."),
			Type:        domain.TypeDecision,
			Scope:       domain.ScopeGlobal,
			Source:      "integration-test",
			Tags:        []string{"storage"},
			Metadata:    map[string]any{},
			Confidence:  1,
		},
		{
			Content:     "Memora project memory search uses tsvector.",
			ContentHash: domain.SHA256Hash("memora project memory search uses tsvector."),
			Type:        domain.TypeFact,
			Scope:       domain.ScopeProject,
			Project:     "memora",
			Source:      "integration-test",
			Tags:        []string{"search", "storage"},
			Metadata:    map[string]any{},
			Confidence:  0.95,
		},
		{
			Content:     "Other project memory should stay isolated.",
			ContentHash: domain.SHA256Hash("other project memory should stay isolated."),
			Type:        domain.TypeFact,
			Scope:       domain.ScopeProject,
			Project:     "other",
			Source:      "integration-test",
			Tags:        []string{"search"},
			Metadata:    map[string]any{},
			Confidence:  0.9,
		},
	}

	require.NoError(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		for _, memory := range memories {
			id, err := repo.InsertMemory(ctx, tx, memory)
			if err != nil {
				return err
			}
			memory.ID = id
		}
		return nil
	}))

	listed, cursor, total, err := repo.ListMemories(ctx, port.MemoryFilter{Scope: domain.ScopeProject, Project: "memora", IncludeGlobal: true, Limit: 10})
	require.NoError(t, err)
	require.Empty(t, cursor)
	require.Equal(t, 2, total)
	require.Len(t, listed, 2)
	require.ElementsMatch(t, []string{memories[0].ID, memories[1].ID}, []string{listed[0].ID, listed[1].ID})

	projectOnly, _, total, err := repo.ListMemories(ctx, port.MemoryFilter{Scope: domain.ScopeProject, Project: "memora", Limit: 10})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, projectOnly, 1)
	require.Equal(t, memories[1].ID, projectOnly[0].ID)

	results, ranks, err := repo.SearchKeyword(ctx, "tsvector", port.MemoryFilter{Scope: domain.ScopeProject, Project: "memora", IncludeGlobal: true, Limit: 10})
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Len(t, ranks, 1)
	require.Equal(t, memories[1].ID, results[0].ID)
	require.Greater(t, ranks[0], 0.0)

	totalCount, activeCount, byType, err := repo.CountMemories(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, totalCount)
	require.Equal(t, 3, activeCount)
	require.Equal(t, 2, byType[string(domain.TypeFact)])
	require.Equal(t, 1, byType[string(domain.TypeDecision)])
}

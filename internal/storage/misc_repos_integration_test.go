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

func TestAuditTokenAndReindexRepositories(t *testing.T) {
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
	memories := NewMemoryRepository(pool)
	profiles := NewEmbeddingProfileRepository(pool)
	audit := NewAuditRepository(pool)
	tokens := NewConfirmationTokenRepository(pool)
	jobs := NewReindexJobRepository(pool)

	memory := &domain.Memory{
		Content:     "Token and audit repository memory.",
		ContentHash: domain.SHA256Hash("token and audit repository memory."),
		Type:        domain.TypeFact,
		Scope:       domain.ScopeProject,
		Project:     "memora",
		Source:      "integration-test",
		Tags:        []string{},
		Metadata:    map[string]any{},
		Confidence:  1,
	}
	profile := &domain.EmbeddingProfile{
		Name:           "reindex-profile",
		Provider:       domain.ProviderOpenAI,
		Model:          "text-embedding-3-small",
		Dimensions:     3,
		DistanceMetric: domain.DistanceCosine,
	}

	require.NoError(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		memoryID, err := memories.InsertMemory(ctx, tx, memory)
		require.NoError(t, err)
		memory.ID = memoryID

		profileID, err := profiles.InsertProfile(ctx, tx, profile)
		require.NoError(t, err)
		profile.ID = profileID

		return audit.InsertAuditEntry(ctx, tx, &domain.AuditEntry{
			Action:     domain.ActionRemember,
			MemoryID:   memory.ID,
			Source:     "integration-test",
			Scope:      domain.ScopeProject,
			Project:    "memora",
			Details:    map[string]any{"result": "stored"},
			DurationMS: 12,
			Success:    true,
		})
	}))

	expiresAt := time.Now().Add(time.Hour)
	var tokenID string
	require.NoError(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		tokenID, err = tokens.CreateToken(ctx, tx, memory.ID, expiresAt)
		return err
	}))
	require.NotEmpty(t, tokenID)

	token, err := tokens.GetToken(ctx, tokenID)
	require.NoError(t, err)
	require.Equal(t, memory.ID, token.MemoryID)
	require.Nil(t, token.UsedAt)

	require.NoError(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		return tokens.ConsumeToken(ctx, tx, tokenID, memory.ID)
	}))
	token, err = tokens.GetToken(ctx, tokenID)
	require.NoError(t, err)
	require.NotNil(t, token.UsedAt)

	var invalidToken *domain.TokenInvalidError
	require.ErrorAs(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		return tokens.ConsumeToken(ctx, tx, tokenID, memory.ID)
	}), &invalidToken)

	var jobID string
	require.NoError(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		jobID, err = jobs.CreateJob(ctx, tx, &domain.ReindexJob{ProfileID: profile.ID, Project: "memora", TotalCount: intPtr(10)})
		return err
	}))
	require.NotEmpty(t, jobID)

	running, err := jobs.GetRunningJob(ctx, profile.ID)
	require.NoError(t, err)
	require.Equal(t, jobID, running.ID)
	require.Equal(t, "memora", running.Project)
	require.Equal(t, 10, *running.TotalCount)

	require.NoError(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		return jobs.UpdateJobProgress(ctx, tx, jobID, 3, 1, 0, memory.ID)
	}))
	job, err := jobs.GetJobByID(ctx, jobID)
	require.NoError(t, err)
	require.Equal(t, 3, job.ProcessedCount)
	require.Equal(t, 1, job.SkippedCount)
	require.Equal(t, memory.ID, job.LastMemoryID)

	require.NoError(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		return jobs.CompleteJob(ctx, tx, jobID)
	}))
	job, err = jobs.GetJobByID(ctx, jobID)
	require.NoError(t, err)
	require.Equal(t, domain.ReindexJobCompleted, job.Status)
	require.NotNil(t, job.CompletedAt)
}

func intPtr(value int) *int {
	return &value
}

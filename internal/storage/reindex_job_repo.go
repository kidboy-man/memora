package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kidboy-man/memora/internal/domain"
	"github.com/kidboy-man/memora/internal/port"
)

type ReindexJobRepository struct {
	pool *pgxpool.Pool
}

func NewReindexJobRepository(pool *pgxpool.Pool) *ReindexJobRepository {
	return &ReindexJobRepository{pool: pool}
}

func (r *ReindexJobRepository) CreateJob(ctx context.Context, tx port.Tx, job *domain.ReindexJob) (string, error) {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return "", err
	}

	project := any(nil)
	if job.Project != "" {
		project = job.Project
	}

	var id string
	err = pgxTx.QueryRow(ctx, `
		INSERT INTO reindex_jobs (profile_id, project, total_count)
		VALUES ($1, $2, $3)
		RETURNING id
	`, job.ProfileID, project, job.TotalCount).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create reindex job: %w", err)
	}
	return id, nil
}

func (r *ReindexJobRepository) UpdateJobProgress(ctx context.Context, tx port.Tx, jobID string, processedCount int, skippedCount int, errorCount int, lastMemoryID string) error {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return err
	}

	lastID := any(nil)
	if lastMemoryID != "" {
		lastID = lastMemoryID
	}
	commandTag, err := pgxTx.Exec(ctx, `
		UPDATE reindex_jobs
		SET processed_count = $2, skipped_count = $3, error_count = $4, last_memory_id = $5
		WHERE id = $1
	`, jobID, processedCount, skippedCount, errorCount, lastID)
	if err != nil {
		return fmt.Errorf("update reindex job progress: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return &domain.NotFoundError{Resource: "reindex job", ID: jobID}
	}
	return nil
}

func (r *ReindexJobRepository) CompleteJob(ctx context.Context, tx port.Tx, jobID string) error {
	return r.finishJob(ctx, tx, jobID, domain.ReindexJobCompleted)
}

func (r *ReindexJobRepository) FailJob(ctx context.Context, tx port.Tx, jobID string) error {
	return r.finishJob(ctx, tx, jobID, domain.ReindexJobFailed)
}

func (r *ReindexJobRepository) GetRunningJob(ctx context.Context, profileID string) (*domain.ReindexJob, error) {
	job, err := scanReindexJob(r.pool.QueryRow(ctx, `
		SELECT id, profile_id, coalesce(project, ''), status, total_count, processed_count, skipped_count, error_count,
		       coalesce(last_memory_id::text, ''), started_at, completed_at
		FROM reindex_jobs
		WHERE profile_id = $1 AND status = 'running'
		ORDER BY started_at DESC
		LIMIT 1
	`, profileID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: "running reindex job", ID: profileID}
		}
		return nil, fmt.Errorf("get running reindex job: %w", err)
	}
	return job, nil
}

func (r *ReindexJobRepository) GetJobByID(ctx context.Context, id string) (*domain.ReindexJob, error) {
	job, err := scanReindexJob(r.pool.QueryRow(ctx, `
		SELECT id, profile_id, coalesce(project, ''), status, total_count, processed_count, skipped_count, error_count,
		       coalesce(last_memory_id::text, ''), started_at, completed_at
		FROM reindex_jobs
		WHERE id = $1
	`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: "reindex job", ID: id}
		}
		return nil, fmt.Errorf("get reindex job: %w", err)
	}
	return job, nil
}

func (r *ReindexJobRepository) finishJob(ctx context.Context, tx port.Tx, jobID string, status domain.ReindexJobStatus) error {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return err
	}

	commandTag, err := pgxTx.Exec(ctx, `
		UPDATE reindex_jobs
		SET status = $2, completed_at = NOW()
		WHERE id = $1
	`, jobID, status)
	if err != nil {
		return fmt.Errorf("finish reindex job: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return &domain.NotFoundError{Resource: "reindex job", ID: jobID}
	}
	return nil
}

func scanReindexJob(row pgx.Row) (*domain.ReindexJob, error) {
	var job domain.ReindexJob
	err := row.Scan(
		&job.ID,
		&job.ProfileID,
		&job.Project,
		&job.Status,
		&job.TotalCount,
		&job.ProcessedCount,
		&job.SkippedCount,
		&job.ErrorCount,
		&job.LastMemoryID,
		&job.StartedAt,
		&job.CompletedAt,
	)
	if err != nil {
		return nil, err
	}
	return &job, nil
}

var _ port.ReindexJobRepository = (*ReindexJobRepository)(nil)

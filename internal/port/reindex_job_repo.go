package port

import (
	"context"

	"github.com/kidboy-man/memora/internal/domain"
)

type ReindexJobRepository interface {
	CreateJob(ctx context.Context, tx Tx, job *domain.ReindexJob) (string, error)
	UpdateJobProgress(ctx context.Context, tx Tx, jobID string, processedCount int, skippedCount int, errorCount int, lastMemoryID string) error
	CompleteJob(ctx context.Context, tx Tx, jobID string) error
	FailJob(ctx context.Context, tx Tx, jobID string) error
	GetRunningJob(ctx context.Context, profileID string) (*domain.ReindexJob, error)
	GetJobByID(ctx context.Context, id string) (*domain.ReindexJob, error)
}

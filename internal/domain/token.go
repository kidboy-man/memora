package domain

import "time"

type ConfirmationToken struct {
	ID        string
	MemoryID  string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

type ReindexJobStatus string

const (
	ReindexJobRunning   ReindexJobStatus = "running"
	ReindexJobCompleted ReindexJobStatus = "completed"
	ReindexJobFailed    ReindexJobStatus = "failed"
)

type ReindexJob struct {
	ID             string
	ProfileID      string
	Project        string
	Status         ReindexJobStatus
	TotalCount     *int
	ProcessedCount int
	SkippedCount   int
	ErrorCount     int
	LastMemoryID   string
	StartedAt      time.Time
	CompletedAt    *time.Time
}

package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kidboy-man/memora/internal/domain"
	"github.com/kidboy-man/memora/internal/port"
)

type AuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

func (r *AuditRepository) InsertAuditEntry(ctx context.Context, tx port.Tx, entry *domain.AuditEntry) error {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return err
	}

	details, err := json.Marshal(entry.Details)
	if err != nil {
		return fmt.Errorf("marshal audit details: %w", err)
	}

	memoryID := any(nil)
	if entry.MemoryID != "" {
		memoryID = entry.MemoryID
	}
	project := any(nil)
	if entry.Project != "" {
		project = entry.Project
	}
	durationMS := any(nil)
	if entry.DurationMS > 0 {
		durationMS = entry.DurationMS
	}
	errorMsg := any(nil)
	if entry.ErrorMsg != "" {
		errorMsg = entry.ErrorMsg
	}

	_, err = pgxTx.Exec(ctx, `
		INSERT INTO audit_log (action, memory_id, source, scope, project, details, duration_ms, success, error_msg)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, entry.Action, memoryID, entry.Source, entry.Scope, project, details, durationMS, entry.Success, errorMsg)
	if err != nil {
		return fmt.Errorf("insert audit entry: %w", err)
	}
	return nil
}

var _ port.AuditRepository = (*AuditRepository)(nil)

package port

import (
	"context"

	"github.com/kidboy-man/memora/internal/domain"
)

type AuditRepository interface {
	InsertAuditEntry(ctx context.Context, tx Tx, entry *domain.AuditEntry) error
}

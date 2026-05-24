package port

import (
	"context"
	"time"

	"github.com/kidboy-man/memora/internal/domain"
)

type ConfirmationTokenRepository interface {
	CreateToken(ctx context.Context, tx Tx, memoryID string, expiresAt time.Time) (string, error)
	ConsumeToken(ctx context.Context, tx Tx, tokenID string, memoryID string) error
	CleanupExpiredTokens(ctx context.Context) error
	GetToken(ctx context.Context, tokenID string) (*domain.ConfirmationToken, error)
}

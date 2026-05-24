package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kidboy-man/memora/internal/domain"
	"github.com/kidboy-man/memora/internal/port"
)

type ConfirmationTokenRepository struct {
	pool *pgxpool.Pool
}

func NewConfirmationTokenRepository(pool *pgxpool.Pool) *ConfirmationTokenRepository {
	return &ConfirmationTokenRepository{pool: pool}
}

func (r *ConfirmationTokenRepository) CreateToken(ctx context.Context, tx port.Tx, memoryID string, expiresAt time.Time) (string, error) {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return "", err
	}

	var id string
	err = pgxTx.QueryRow(ctx, `
		INSERT INTO confirmation_tokens (memory_id, expires_at)
		VALUES ($1, $2)
		RETURNING id
	`, memoryID, expiresAt).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create confirmation token: %w", err)
	}
	return id, nil
}

func (r *ConfirmationTokenRepository) ConsumeToken(ctx context.Context, tx port.Tx, tokenID string, memoryID string) error {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return err
	}

	commandTag, err := pgxTx.Exec(ctx, `
		UPDATE confirmation_tokens
		SET used_at = NOW()
		WHERE id = $1 AND memory_id = $2 AND used_at IS NULL AND expires_at > NOW()
	`, tokenID, memoryID)
	if err != nil {
		return fmt.Errorf("consume confirmation token: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return &domain.TokenInvalidError{}
	}
	return nil
}

func (r *ConfirmationTokenRepository) CleanupExpiredTokens(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM confirmation_tokens WHERE expires_at < NOW()`)
	if err != nil {
		return fmt.Errorf("cleanup expired confirmation tokens: %w", err)
	}
	return nil
}

func (r *ConfirmationTokenRepository) GetToken(ctx context.Context, tokenID string) (*domain.ConfirmationToken, error) {
	var token domain.ConfirmationToken
	err := r.pool.QueryRow(ctx, `
		SELECT id, memory_id, expires_at, used_at, created_at
		FROM confirmation_tokens
		WHERE id = $1
	`, tokenID).Scan(&token.ID, &token.MemoryID, &token.ExpiresAt, &token.UsedAt, &token.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: "confirmation token", ID: tokenID}
		}
		return nil, fmt.Errorf("get confirmation token: %w", err)
	}
	return &token, nil
}

var _ port.ConfirmationTokenRepository = (*ConfirmationTokenRepository)(nil)

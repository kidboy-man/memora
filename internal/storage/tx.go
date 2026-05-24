package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kidboy-man/memora/internal/port"
)

type PgxTxRunner struct {
	pool *pgxpool.Pool
}

func NewTxRunner(pool *pgxpool.Pool) *PgxTxRunner {
	return &PgxTxRunner{pool: pool}
}

func (r *PgxTxRunner) WithinTx(ctx context.Context, fn func(context.Context, port.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	wrapped := pgxTx{tx: tx}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	if err := fn(ctx, wrapped); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	committed = true
	return nil
}

type pgxTx struct {
	tx pgx.Tx
}

func (pgxTx) IsTx() {}

func unwrapTx(tx port.Tx) (pgx.Tx, error) {
	wrapped, ok := tx.(pgxTx)
	if !ok {
		return nil, fmt.Errorf("unsupported transaction type %T", tx)
	}
	if wrapped.tx == nil {
		return nil, errors.New("nil transaction")
	}
	return wrapped.tx, nil
}

var _ port.TxRunner = (*PgxTxRunner)(nil)

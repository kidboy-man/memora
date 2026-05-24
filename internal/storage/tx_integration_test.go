//go:build integration

package storage

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kidboy-man/memora/internal/config"
	"github.com/kidboy-man/memora/internal/port"
	"github.com/stretchr/testify/require"
)

func configFromDSN(dsn string) (config.DatabaseConfig, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return config.DatabaseConfig{}, err
	}

	password, _ := u.User.Password()
	host := u.Hostname()
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return config.DatabaseConfig{}, err
	}

	return config.DatabaseConfig{
		Host:     host,
		Port:     port,
		Name:     strings.TrimPrefix(u.Path, "/"),
		User:     u.User.Username(),
		Password: password,
		SSLMode:  u.Query().Get("sslmode"),
	}, nil
}

func TestTxRunnerWithinTxCommitsAndRollsBack(t *testing.T) {
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

	require.NoError(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		pgxTx, err := unwrapTx(tx)
		require.NoError(t, err)
		_, err = pgxTx.Exec(ctx, `
			INSERT INTO embedding_profiles (name, provider, model, dimensions, is_active)
			VALUES ('commit-profile', 'openai', 'text-embedding-3-small', 1536, TRUE)
		`)
		return err
	}))

	var committed int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM embedding_profiles WHERE name = 'commit-profile'`).Scan(&committed))
	require.Equal(t, 1, committed)

	rollbackErr := errors.New("force rollback")
	require.ErrorIs(t, runner.WithinTx(ctx, func(ctx context.Context, tx port.Tx) error {
		pgxTx, err := unwrapTx(tx)
		require.NoError(t, err)
		_, err = pgxTx.Exec(ctx, `
			INSERT INTO embedding_profiles (name, provider, model, dimensions)
			VALUES ('rollback-profile', 'openai', 'text-embedding-3-small', 1536)
		`)
		require.NoError(t, err)
		return rollbackErr
	}), rollbackErr)

	var rolledBack int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM embedding_profiles WHERE name = 'rollback-profile'`).Scan(&rolledBack))
	require.Equal(t, 0, rolledBack)
}

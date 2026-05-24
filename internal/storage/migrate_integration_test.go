//go:build integration

package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestMigrations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	dsn := startPostgres(t, ctx)

	require.NoError(t, RunMigrations(ctx, dsn, DirectionUp))
	requireTablesExist(t, ctx, dsn, []string{
		"schema_migrations",
		"embedding_profiles",
		"memories",
		"memory_embeddings",
		"audit_log",
		"confirmation_tokens",
		"reindex_jobs",
	})
	requireSchemaInvariants(t, ctx, dsn)

	require.NoError(t, RunMigrations(ctx, dsn, DirectionUp))

	require.NoError(t, RunMigrations(ctx, dsn, DirectionDown))
	requireTablesAbsent(t, ctx, dsn, []string{
		"embedding_profiles",
		"memories",
		"memory_embeddings",
		"audit_log",
		"confirmation_tokens",
		"reindex_jobs",
	})
}

func startPostgres(t *testing.T, ctx context.Context) string {
	t.Helper()

	container, err := postgres.Run(ctx,
		"pgvector/pgvector:pg16",
		postgres.WithDatabase("memora"),
		postgres.WithUsername("memora"),
		postgres.WithPassword("memora_dev_password"),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, testcontainers.TerminateContainer(container))
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	waitForPostgres(t, ctx, dsn)
	return dsn
}

func waitForPostgres(t *testing.T, ctx context.Context, dsn string) {
	t.Helper()

	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		db, err := sql.Open("pgx", dsn)
		require.NoError(collect, err)
		defer db.Close()

		require.NoError(collect, db.PingContext(ctx))
	}, 30*time.Second, 250*time.Millisecond)
}

func requireTablesExist(t *testing.T, ctx context.Context, dsn string, tables []string) {
	t.Helper()

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer db.Close()

	for _, table := range tables {
		var exists bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = 'public'
				  AND table_name = $1
			)`, table).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "expected table %s to exist", table)
	}
}

func requireTablesAbsent(t *testing.T, ctx context.Context, dsn string, tables []string) {
	t.Helper()

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer db.Close()

	for _, table := range tables {
		var exists bool
		err := db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = 'public'
				  AND table_name = $1
			)`, table).Scan(&exists)
		require.NoError(t, err)
		require.Falsef(t, exists, "expected table %s to be absent", table)
	}
}

func requireSchemaInvariants(t *testing.T, ctx context.Context, dsn string) {
	t.Helper()

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	defer db.Close()

	var embeddingType string
	err = db.QueryRowContext(ctx, `
		SELECT udt_name
		FROM information_schema.columns
		WHERE table_schema = 'public'
		  AND table_name = 'memory_embeddings'
		  AND column_name = 'embedding'
	`).Scan(&embeddingType)
	require.NoError(t, err)
	require.Equal(t, "vector", embeddingType)

	_, err = db.ExecContext(ctx, `
		INSERT INTO memories (content, content_hash, type, scope, source, confidence)
		VALUES ('zero confidence', decode(repeat('00', 32), 'hex'), 'fact', 'global', 'test', 0.0)
	`)
	require.Error(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO memories (content, content_hash, type, scope, source, confidence)
		VALUES ('global duplicate one', decode(repeat('01', 32), 'hex'), 'fact', 'global', 'test', 1.0)
	`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO memories (content, content_hash, type, scope, source, confidence)
		VALUES ('global duplicate two', decode(repeat('01', 32), 'hex'), 'fact', 'global', 'test', 1.0)
	`)
	require.Error(t, err)

	var memoryID string
	err = db.QueryRowContext(ctx, `
		INSERT INTO memories (content, content_hash, type, scope, source, confidence)
		VALUES ('token memory', decode(repeat('02', 32), 'hex'), 'fact', 'global', 'test', 1.0)
		RETURNING id
	`).Scan(&memoryID)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO confirmation_tokens (memory_id, expires_at)
		VALUES ($1, NOW() + INTERVAL '5 minutes')
	`, memoryID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO confirmation_tokens (memory_id, expires_at)
		VALUES ($1, NOW() + INTERVAL '5 minutes')
	`, memoryID)
	require.Error(t, err)
}

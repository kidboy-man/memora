package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	pgxv5 "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type Direction string

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
)

func RunMigrations(ctx context.Context, dsn string, direction Direction) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("migration context: %w", err)
	}
	if dsn == "" {
		return fmt.Errorf("migration dsn is required")
	}
	if direction != DirectionUp && direction != DirectionDown {
		return fmt.Errorf("unknown migration direction %q", direction)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open migration database: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping migration database: %w", err)
	}

	sourceDriver, err := iofs.New(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("load migration files: %w", err)
	}

	databaseDriver, err := pgxv5.WithInstance(db, &pgxv5.Config{})
	if err != nil {
		return fmt.Errorf("create migration database driver: %w", err)
	}

	migrator, err := migrate.NewWithInstance("iofs", sourceDriver, "pgx5", databaseDriver)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer migrator.Close()

	if version, dirty, err := migrator.Version(); err == nil && dirty {
		return fmt.Errorf("migration version %d is dirty", version)
	} else if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("read migration version: %w", err)
	}

	switch direction {
	case DirectionUp:
		if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("migrate up: %w", err)
		}
	case DirectionDown:
		if err := migrator.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("migrate down: %w", err)
		}
	}

	return nil
}

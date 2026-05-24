package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunMigrationsRejectsInvalidDirection(t *testing.T) {
	err := RunMigrations(context.Background(), "postgres://user:pass@localhost:5432/memora?sslmode=disable", Direction("sideways"))

	require.ErrorContains(t, err, `unknown migration direction "sideways"`)
}

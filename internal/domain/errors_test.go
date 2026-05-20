package domain_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kidboy-man/memora/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		contain string
	}{
		{
			name:    "ValidationError single field",
			err:     &domain.ValidationError{Fields: []string{"content must not be empty"}},
			contain: "content must not be empty",
		},
		{
			name:    "ValidationError multiple fields",
			err:     &domain.ValidationError{Fields: []string{"content must not be empty", "source is required"}},
			contain: "source is required",
		},
		{
			name:    "NotFoundError",
			err:     &domain.NotFoundError{Resource: "memory", ID: "abc-123"},
			contain: "abc-123",
		},
		{
			name:    "VersionConflictError",
			err:     &domain.VersionConflictError{Expected: 3, Actual: 5},
			contain: "3",
		},
		{
			name:    "ExactDuplicateError",
			err:     &domain.ExactDuplicateError{ExistingID: "dup-id"},
			contain: "dup-id",
		},
		{
			name:    "SimilarDuplicateError",
			err:     &domain.SimilarDuplicateError{ExistingID: "sim-id", Score: 0.95},
			contain: "sim-id",
		},
		{
			name:    "EmbeddingFailedError",
			err:     &domain.EmbeddingFailedError{Cause: fmt.Errorf("timeout")},
			contain: "timeout",
		},
		{
			name:    "TokenInvalidError",
			err:     &domain.TokenInvalidError{},
			contain: "token",
		},
		{
			name:    "AlreadyDeletedError",
			err:     &domain.AlreadyDeletedError{MemoryID: "mem-xyz"},
			contain: "mem-xyz",
		},
		{
			name:    "NoActiveProfileError",
			err:     &domain.NoActiveProfileError{},
			contain: "no active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, tt.err.Error(), tt.contain)
		})
	}
}

func TestErrorsAs(t *testing.T) {
	t.Run("ValidationError unwrappable", func(t *testing.T) {
		err := fmt.Errorf("wrap: %w", &domain.ValidationError{Fields: []string{"x"}})
		var ve *domain.ValidationError
		require.ErrorAs(t, err, &ve)
	})

	t.Run("EmbeddingFailedError unwraps cause", func(t *testing.T) {
		cause := fmt.Errorf("provider down")
		err := &domain.EmbeddingFailedError{Cause: cause}
		assert.ErrorIs(t, err, cause)
	})

	t.Run("NotFoundError distinct from ValidationError", func(t *testing.T) {
		err := &domain.NotFoundError{Resource: "memory", ID: "x"}
		var ve *domain.ValidationError
		assert.False(t, errors.As(err, &ve))
	})
}

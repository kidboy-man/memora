package domain_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kidboy-man/memora/internal/domain"
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
			msg := tt.err.Error()
			if !containsSubstring(msg, tt.contain) {
				t.Errorf("error message %q does not contain %q", msg, tt.contain)
			}
		})
	}
}

func TestErrorsAs(t *testing.T) {
	t.Run("ValidationError unwrappable", func(t *testing.T) {
		err := fmt.Errorf("wrap: %w", &domain.ValidationError{Fields: []string{"x"}})
		var ve *domain.ValidationError
		if !errors.As(err, &ve) {
			t.Error("errors.As should find ValidationError through wrapping")
		}
	})

	t.Run("EmbeddingFailedError unwraps cause", func(t *testing.T) {
		cause := fmt.Errorf("provider down")
		err := &domain.EmbeddingFailedError{Cause: cause}
		if !errors.Is(err, cause) {
			t.Error("errors.Is should find cause through EmbeddingFailedError.Unwrap()")
		}
	})

	t.Run("NotFoundError distinct from ValidationError", func(t *testing.T) {
		err := &domain.NotFoundError{Resource: "memory", ID: "x"}
		var ve *domain.ValidationError
		if errors.As(err, &ve) {
			t.Error("NotFoundError should not match ValidationError")
		}
	})
}

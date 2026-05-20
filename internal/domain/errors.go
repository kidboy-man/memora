package domain

import (
	"fmt"
	"strings"
)

// ValidationError is returned when input fails domain validation.
// Fields lists every failing constraint so callers get all errors in one pass.
type ValidationError struct {
	Fields []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s", strings.Join(e.Fields, "; "))
}

// NotFoundError is returned when a memory or resource does not exist.
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s %q not found", e.Resource, e.ID)
}

// VersionConflictError is returned when expected_version doesn't match current.
type VersionConflictError struct {
	Expected int
	Actual   int
}

func (e *VersionConflictError) Error() string {
	return fmt.Sprintf("version conflict: expected %d, actual %d", e.Expected, e.Actual)
}

// ExactDuplicateError is returned when content hash already exists within scope/project/type.
type ExactDuplicateError struct {
	ExistingID string
}

func (e *ExactDuplicateError) Error() string {
	return fmt.Sprintf("exact duplicate of memory %q", e.ExistingID)
}

// SimilarDuplicateError is returned when similarity dedup triggers (strategy=skip or warn).
type SimilarDuplicateError struct {
	ExistingID string
	Score      float64
}

func (e *SimilarDuplicateError) Error() string {
	return fmt.Sprintf("similar duplicate of memory %q (score %.4f)", e.ExistingID, e.Score)
}

// EmbeddingFailedError wraps the underlying provider error.
type EmbeddingFailedError struct {
	Cause error
}

func (e *EmbeddingFailedError) Error() string {
	return fmt.Sprintf("embedding failed: %v", e.Cause)
}

func (e *EmbeddingFailedError) Unwrap() error { return e.Cause }

// TokenInvalidError is returned when a confirmation token is invalid, expired, or already used.
type TokenInvalidError struct{}

func (e *TokenInvalidError) Error() string {
	return "confirmation token invalid, expired, or already used"
}

// AlreadyDeletedError is returned when trying to delete an already-deleted memory.
type AlreadyDeletedError struct {
	MemoryID string
}

func (e *AlreadyDeletedError) Error() string {
	return fmt.Sprintf("memory %q is already deleted", e.MemoryID)
}

// NoActiveProfileError is returned when no active embedding profile is configured.
type NoActiveProfileError struct{}

func (e *NoActiveProfileError) Error() string {
	return "no active embedding profile configured"
}

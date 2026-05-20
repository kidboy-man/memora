package domain

import (
	"fmt"
	"strings"
	"time"
)

// MemoryType classifies what kind of knowledge a memory represents.
type MemoryType string

// Scope controls memory visibility: global (all projects) or project-scoped.
type Scope string

// DedupeStrategy controls behaviour when a similar memory already exists.
type DedupeStrategy string

const (
	TypeFact           MemoryType = "fact"
	TypeDecision       MemoryType = "decision"
	TypePreference     MemoryType = "preference"
	TypeProjectContext MemoryType = "project_context"
)

const (
	ScopeGlobal  Scope = "global"
	ScopeProject Scope = "project"
)

const (
	DedupeNone DedupeStrategy = "none"
	DedupeWarn DedupeStrategy = "warn"
	DedupeSkip DedupeStrategy = "skip"
)

var validMemoryTypes = map[MemoryType]bool{
	TypeFact:           true,
	TypeDecision:       true,
	TypePreference:     true,
	TypeProjectContext: true,
}

// Memory is the core domain entity: an atomic, self-contained piece of knowledge
// that agents can remember, recall, update, and forget.
type Memory struct {
	ID           string
	Content      string
	ContentHash  []byte // SHA-256 of NormalizeContent(Content); stored as BYTEA
	Type         MemoryType
	Scope        Scope
	Project      string // empty when Scope=global
	Source       string // agent or process that created this memory
	Tags         []string
	Metadata     map[string]any
	Confidence   float64 // (0.0, 1.0]; service defaults to 1.0 when agent omits
	Version      int
	DeletedAt    *time.Time
	DeletedBy    string
	DeleteReason string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Validate checks all domain invariants for a Memory. It collects every
// violation and returns them together so callers get a complete picture
// in a single round-trip.
func (m *Memory) Validate() error {
	var errs []string

	if strings.TrimSpace(m.Content) == "" {
		errs = append(errs, "content must not be empty")
	}
	if !validMemoryTypes[m.Type] {
		errs = append(errs, fmt.Sprintf("invalid type %q", m.Type))
	}
	if m.Scope != ScopeGlobal && m.Scope != ScopeProject {
		errs = append(errs, fmt.Sprintf("invalid scope %q", m.Scope))
	}
	if m.Scope == ScopeProject && strings.TrimSpace(m.Project) == "" {
		errs = append(errs, "project is required for project scope")
	}
	// Confidence must be in (0.0, 1.0]. Zero is excluded: it would be
	// indistinguishable from the Go zero value and has no meaningful semantics.
	if m.Confidence <= 0.0 || m.Confidence > 1.0 {
		errs = append(errs, fmt.Sprintf("confidence %.3f out of range (0.0, 1.0]", m.Confidence))
	}
	if strings.TrimSpace(m.Source) == "" {
		errs = append(errs, "source is required")
	}

	if len(errs) > 0 {
		return &ValidationError{Fields: errs}
	}
	return nil
}

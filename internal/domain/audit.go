package domain

import "time"

// AuditAction identifies the operation recorded in an audit log entry.
type AuditAction string

const (
	ActionRemember    AuditAction = "remember"
	ActionRecall      AuditAction = "recall"
	ActionUpdate      AuditAction = "update"
	ActionForget      AuditAction = "forget"
	ActionHealthCheck AuditAction = "health_check"
	ActionList        AuditAction = "list"
	ActionGetContext  AuditAction = "get_context"
)

// AuditEntry records a single committed operation. It is written inside the
// same transaction as the originating write (remember, update, forget) to
// guarantee log integrity. Read-only operations (recall, list, get_context,
// health_check) are logged outside a transaction.
type AuditEntry struct {
	ID         string
	OccurredAt time.Time
	Action     AuditAction
	MemoryID   string // empty when the action is not memory-specific
	Source     string
	Scope      Scope
	Project    string
	Details    map[string]any // arbitrary context; e.g. dedup strategy, similarity score
	DurationMS int64
	Success    bool
	ErrorMsg   string // empty on success
}

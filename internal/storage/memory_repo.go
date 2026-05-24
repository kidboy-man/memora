package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kidboy-man/memora/internal/domain"
	"github.com/kidboy-man/memora/internal/port"
	"github.com/pgvector/pgvector-go"
)

type MemoryRepository struct {
	pool *pgxpool.Pool
}

func NewMemoryRepository(pool *pgxpool.Pool) *MemoryRepository {
	return &MemoryRepository{pool: pool}
}

func (r *MemoryRepository) InsertMemory(ctx context.Context, tx port.Tx, memory *domain.Memory) (string, error) {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return "", err
	}

	metadata, err := json.Marshal(memory.Metadata)
	if err != nil {
		return "", fmt.Errorf("marshal memory metadata: %w", err)
	}

	project := projectParam(memory.Scope, memory.Project)
	var id string
	err = pgxTx.QueryRow(ctx, `
		INSERT INTO memories (content, content_hash, type, scope, project, source, tags, metadata, confidence)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, memory.Content, memory.ContentHash, memory.Type, memory.Scope, project, memory.Source, memory.Tags, metadata, memory.Confidence).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return "", &domain.ExactDuplicateError{}
		}
		return "", fmt.Errorf("insert memory: %w", err)
	}
	return id, nil
}

func (r *MemoryRepository) GetMemoryByID(ctx context.Context, id string, includeDeleted bool) (*domain.Memory, error) {
	query := `
		SELECT id, content, content_hash, type, scope, coalesce(project, ''), source, tags, metadata, confidence,
		       version, deleted_at, coalesce(deleted_by, ''), coalesce(delete_reason, ''), created_at, updated_at
		FROM memories
		WHERE id = $1`
	if !includeDeleted {
		query += ` AND deleted_at IS NULL`
	}

	memory, err := scanMemory(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: "memory", ID: id}
		}
		return nil, fmt.Errorf("get memory: %w", err)
	}
	return memory, nil
}

func (r *MemoryRepository) ExistsExact(ctx context.Context, scope domain.Scope, project string, memType domain.MemoryType, contentHash []byte) (bool, string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		SELECT id
		FROM memories
		WHERE scope = $1
		  AND coalesce(project, '') = $2
		  AND type = $3
		  AND content_hash = $4
		  AND deleted_at IS NULL
		LIMIT 1
	`, scope, project, memType, contentHash).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil
		}
		return false, "", fmt.Errorf("check exact memory: %w", err)
	}
	return true, id, nil
}

func (r *MemoryRepository) UpdateMemory(ctx context.Context, tx port.Tx, id string, expectedVersion int, updates port.MemoryUpdates) (*domain.Memory, error) {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return nil, err
	}

	sets := []string{"version = version + 1", "updated_at = NOW()"}
	args := []any{id, expectedVersion}
	if updates.Content != nil {
		args = append(args, *updates.Content)
		sets = append(sets, fmt.Sprintf("content = $%d", len(args)))
	}
	if updates.ContentHash != nil {
		args = append(args, updates.ContentHash)
		sets = append(sets, fmt.Sprintf("content_hash = $%d", len(args)))
	}
	if updates.Tags != nil {
		args = append(args, *updates.Tags)
		sets = append(sets, fmt.Sprintf("tags = $%d", len(args)))
	}
	if updates.Metadata != nil {
		metadata, err := json.Marshal(*updates.Metadata)
		if err != nil {
			return nil, fmt.Errorf("marshal memory metadata: %w", err)
		}
		args = append(args, metadata)
		sets = append(sets, fmt.Sprintf("metadata = $%d", len(args)))
	}
	if updates.Confidence != nil {
		args = append(args, *updates.Confidence)
		sets = append(sets, fmt.Sprintf("confidence = $%d", len(args)))
	}

	query := fmt.Sprintf(`
		UPDATE memories
		SET %s
		WHERE id = $1 AND version = $2 AND deleted_at IS NULL
		RETURNING id, content, content_hash, type, scope, coalesce(project, ''), source, tags, metadata, confidence,
		          version, deleted_at, coalesce(deleted_by, ''), coalesce(delete_reason, ''), created_at, updated_at
	`, strings.Join(sets, ", "))

	memory, err := scanMemory(pgxTx.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			actual, actualErr := currentVersion(ctx, pgxTx, id)
			if actualErr != nil {
				return nil, actualErr
			}
			return nil, &domain.VersionConflictError{Expected: expectedVersion, Actual: actual}
		}
		return nil, fmt.Errorf("update memory: %w", err)
	}
	return memory, nil
}

func (r *MemoryRepository) SoftDeleteMemory(ctx context.Context, tx port.Tx, id string, deletedBy string, reason string) error {
	pgxTx, err := unwrapTx(tx)
	if err != nil {
		return err
	}

	var alreadyDeleted bool
	err = pgxTx.QueryRow(ctx, `
		UPDATE memories
		SET deleted_at = NOW(), deleted_by = $2, delete_reason = $3, updated_at = NOW(), version = version + 1
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING false
	`, id, deletedBy, reason).Scan(&alreadyDeleted)
	if err == nil {
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("soft delete memory: %w", err)
	}

	var deletedAtSet bool
	err = pgxTx.QueryRow(ctx, `SELECT deleted_at IS NOT NULL FROM memories WHERE id = $1`, id).Scan(&deletedAtSet)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &domain.NotFoundError{Resource: "memory", ID: id}
		}
		return fmt.Errorf("read memory deletion state: %w", err)
	}
	if deletedAtSet {
		return &domain.AlreadyDeletedError{MemoryID: id}
	}
	return &domain.NotFoundError{Resource: "memory", ID: id}
}

func (r *MemoryRepository) FindSimilar(ctx context.Context, profileID string, embedding []float32, threshold float64, scope domain.Scope, project string, memType domain.MemoryType, limit int) ([]port.SimilarResult, error) {
	args := []any{profileID, pgvector.NewVector(embedding), threshold, scope, project, memType, limitOrDefault(limit)}
	rows, err := r.pool.Query(ctx, `
		SELECT m.id, 1 - (me.embedding <=> $2) AS score
		FROM memory_embeddings me
		JOIN memories m ON m.id = me.memory_id
		WHERE me.profile_id = $1
		  AND 1 - (me.embedding <=> $2) >= $3
		  AND m.scope = $4
		  AND coalesce(m.project, '') = $5
		  AND m.type = $6
		  AND m.deleted_at IS NULL
		ORDER BY score DESC, m.created_at DESC
		LIMIT $7
	`, args...)
	if err != nil {
		return nil, fmt.Errorf("find similar memories: %w", err)
	}
	defer rows.Close()

	var results []port.SimilarResult
	for rows.Next() {
		var result port.SimilarResult
		if err := rows.Scan(&result.MemoryID, &result.Score); err != nil {
			return nil, fmt.Errorf("scan similar memory: %w", err)
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate similar memories: %w", err)
	}
	return results, nil
}

func (r *MemoryRepository) SearchSemantic(ctx context.Context, profileID string, embedding []float32, filter port.MemoryFilter) ([]*domain.Memory, []float64, error) {
	clauses, args := memoryFilterClausesWithAlias(filter, "m")
	args = append(args, profileID)
	profileArg := len(args)
	args = append(args, pgvector.NewVector(embedding))
	embeddingArg := len(args)
	args = append(args, limitOrDefault(filter.Limit))
	limitArg := len(args)

	query := fmt.Sprintf(`
		SELECT m.id, m.content, m.content_hash, m.type, m.scope, coalesce(m.project, ''), m.source, m.tags, m.metadata, m.confidence,
		       m.version, m.deleted_at, coalesce(m.deleted_by, ''), coalesce(m.delete_reason, ''), m.created_at, m.updated_at,
		       1 - (me.embedding <=> $%d) AS score
		FROM memory_embeddings me
		JOIN memories m ON m.id = me.memory_id
		WHERE %s
		  AND me.profile_id = $%d
		ORDER BY score DESC, m.created_at DESC
		LIMIT $%d
	`, embeddingArg, strings.Join(clauses, " AND "), profileArg, limitArg)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("search semantic memories: %w", err)
	}
	defer rows.Close()

	var memories []*domain.Memory
	var scores []float64
	for rows.Next() {
		memory, score, err := scanMemoryWithScore(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan semantic memory: %w", err)
		}
		memories = append(memories, memory)
		scores = append(scores, score)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate semantic memories: %w", err)
	}
	return memories, scores, nil
}

func (r *MemoryRepository) SearchKeyword(ctx context.Context, query string, filter port.MemoryFilter) ([]*domain.Memory, []float64, error) {
	clauses, args := memoryFilterClauses(filter)
	args = append(args, query)
	queryArg := len(args)
	args = append(args, limitOrDefault(filter.Limit))
	limitArg := len(args)

	sql := fmt.Sprintf(`
		SELECT id, content, content_hash, type, scope, coalesce(project, ''), source, tags, metadata, confidence,
		       version, deleted_at, coalesce(deleted_by, ''), coalesce(delete_reason, ''), created_at, updated_at,
		       ts_rank(search_vector, plainto_tsquery('simple', $%d)) AS rank
		FROM memories
		WHERE %s
		  AND search_vector @@ plainto_tsquery('simple', $%d)
		ORDER BY rank DESC, created_at DESC
		LIMIT $%d
	`, queryArg, strings.Join(clauses, " AND "), queryArg, limitArg)

	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("search keyword memories: %w", err)
	}
	defer rows.Close()

	var memories []*domain.Memory
	var ranks []float64
	for rows.Next() {
		memory, rank, err := scanMemoryWithScore(rows)
		if err != nil {
			return nil, nil, fmt.Errorf("scan keyword memory: %w", err)
		}
		memories = append(memories, memory)
		ranks = append(ranks, rank)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate keyword memories: %w", err)
	}
	return memories, ranks, nil
}

func (r *MemoryRepository) ListMemories(ctx context.Context, filter port.MemoryFilter) ([]*domain.Memory, string, int, error) {
	clauses, args := memoryFilterClauses(filter)
	countSQL := fmt.Sprintf(`SELECT count(*) FROM memories WHERE %s`, strings.Join(clauses, " AND "))

	var total int
	if err := r.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, "", 0, fmt.Errorf("count filtered memories: %w", err)
	}

	args = append(args, limitOrDefault(filter.Limit))
	limitArg := len(args)
	query := fmt.Sprintf(`
		SELECT id, content, content_hash, type, scope, coalesce(project, ''), source, tags, metadata, confidence,
		       version, deleted_at, coalesce(deleted_by, ''), coalesce(delete_reason, ''), created_at, updated_at
		FROM memories
		WHERE %s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d
	`, strings.Join(clauses, " AND "), limitArg)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list memories: %w", err)
	}
	defer rows.Close()

	var memories []*domain.Memory
	for rows.Next() {
		memory, err := scanMemory(rows)
		if err != nil {
			return nil, "", 0, fmt.Errorf("scan listed memory: %w", err)
		}
		memories = append(memories, memory)
	}
	if err := rows.Err(); err != nil {
		return nil, "", 0, fmt.Errorf("iterate listed memories: %w", err)
	}
	return memories, "", total, nil
}

func (r *MemoryRepository) CountMemories(ctx context.Context) (int, int, map[string]int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM memories`).Scan(&total); err != nil {
		return 0, 0, nil, fmt.Errorf("count memories: %w", err)
	}

	var active int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM memories WHERE deleted_at IS NULL`).Scan(&active); err != nil {
		return 0, 0, nil, fmt.Errorf("count active memories: %w", err)
	}

	rows, err := r.pool.Query(ctx, `SELECT type, count(*) FROM memories WHERE deleted_at IS NULL GROUP BY type`)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("count memories by type: %w", err)
	}
	defer rows.Close()

	byType := make(map[string]int)
	for rows.Next() {
		var memType string
		var count int
		if err := rows.Scan(&memType, &count); err != nil {
			return 0, 0, nil, fmt.Errorf("scan memory type count: %w", err)
		}
		byType[memType] = count
	}
	if err := rows.Err(); err != nil {
		return 0, 0, nil, fmt.Errorf("iterate memory type counts: %w", err)
	}
	return total, active, byType, nil
}

func currentVersion(ctx context.Context, tx pgx.Tx, id string) (int, error) {
	var version int
	err := tx.QueryRow(ctx, `SELECT version FROM memories WHERE id = $1`, id).Scan(&version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, &domain.NotFoundError{Resource: "memory", ID: id}
		}
		return 0, fmt.Errorf("read memory version: %w", err)
	}
	return version, nil
}

func scanMemory(row pgx.Row) (*domain.Memory, error) {
	var memory domain.Memory
	var metadata []byte
	err := row.Scan(
		&memory.ID,
		&memory.Content,
		&memory.ContentHash,
		&memory.Type,
		&memory.Scope,
		&memory.Project,
		&memory.Source,
		&memory.Tags,
		&metadata,
		&memory.Confidence,
		&memory.Version,
		&memory.DeletedAt,
		&memory.DeletedBy,
		&memory.DeleteReason,
		&memory.CreatedAt,
		&memory.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(metadata, &memory.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal memory metadata: %w", err)
	}
	return &memory, nil
}

func scanMemoryWithScore(row pgx.Row) (*domain.Memory, float64, error) {
	var memory domain.Memory
	var metadata []byte
	var score float64
	err := row.Scan(
		&memory.ID,
		&memory.Content,
		&memory.ContentHash,
		&memory.Type,
		&memory.Scope,
		&memory.Project,
		&memory.Source,
		&memory.Tags,
		&metadata,
		&memory.Confidence,
		&memory.Version,
		&memory.DeletedAt,
		&memory.DeletedBy,
		&memory.DeleteReason,
		&memory.CreatedAt,
		&memory.UpdatedAt,
		&score,
	)
	if err != nil {
		return nil, 0, err
	}
	if err := json.Unmarshal(metadata, &memory.Metadata); err != nil {
		return nil, 0, fmt.Errorf("unmarshal memory metadata: %w", err)
	}
	return &memory, score, nil
}

func memoryFilterClauses(filter port.MemoryFilter) ([]string, []any) {
	return memoryFilterClausesWithAlias(filter, "")
}

func memoryFilterClausesWithAlias(filter port.MemoryFilter, alias string) ([]string, []any) {
	col := func(name string) string {
		if alias == "" {
			return name
		}
		return alias + "." + name
	}

	clauses := []string{}
	args := []any{}

	if !filter.IncludeDeleted {
		clauses = append(clauses, col("deleted_at")+" IS NULL")
	}
	if filter.Scope != "" {
		if filter.IncludeGlobal && filter.Scope == domain.ScopeProject {
			args = append(args, domain.ScopeProject)
			projectScopeArg := len(args)
			args = append(args, filter.Project)
			projectArg := len(args)
			args = append(args, domain.ScopeGlobal)
			globalScopeArg := len(args)
			clauses = append(clauses, fmt.Sprintf("((%s = $%d AND %s = $%d) OR %s = $%d)", col("scope"), projectScopeArg, col("project"), projectArg, col("scope"), globalScopeArg))
		} else {
			args = append(args, filter.Scope)
			clauses = append(clauses, fmt.Sprintf("%s = $%d", col("scope"), len(args)))
			if filter.Scope == domain.ScopeProject {
				args = append(args, filter.Project)
				clauses = append(clauses, fmt.Sprintf("%s = $%d", col("project"), len(args)))
			}
		}
	}
	if len(filter.Types) > 0 {
		args = append(args, filter.Types)
		clauses = append(clauses, fmt.Sprintf("%s = ANY($%d)", col("type"), len(args)))
	}
	if len(filter.Tags) > 0 {
		args = append(args, filter.Tags)
		clauses = append(clauses, fmt.Sprintf("%s @> $%d", col("tags"), len(args)))
	}
	if filter.Source != "" {
		args = append(args, filter.Source)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", col("source"), len(args)))
	}
	if len(clauses) == 0 {
		clauses = append(clauses, "true")
	}
	return clauses, args
}

func limitOrDefault(limit int) int {
	if limit <= 0 || limit > 100 {
		return 100
	}
	return limit
}

func projectParam(scope domain.Scope, project string) any {
	if scope == domain.ScopeGlobal {
		return nil
	}
	return project
}

var _ port.MemoryRepository = (*MemoryRepository)(nil)

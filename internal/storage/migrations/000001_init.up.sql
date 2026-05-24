CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "vector";

CREATE FUNCTION memora_text_array_search_text(input_values TEXT[])
RETURNS TEXT
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
AS $$
    SELECT coalesce(array_to_string(input_values, ' '), '')
$$;

CREATE TABLE embedding_profiles (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT        NOT NULL UNIQUE,
    provider        TEXT        NOT NULL CHECK (provider IN ('openrouter', 'openai', 'ollama')),
    model           TEXT        NOT NULL,
    api_base_url    TEXT,
    dimensions      INTEGER     NOT NULL CHECK (dimensions > 0),
    distance_metric TEXT        NOT NULL DEFAULT 'cosine' CHECK (distance_metric IN ('cosine', 'l2', 'ip')),
    is_active       BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX embedding_profiles_one_active
    ON embedding_profiles (is_active) WHERE is_active = TRUE;

CREATE TABLE memories (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    content         TEXT         NOT NULL CHECK (length(content) > 0),
    content_hash    BYTEA        NOT NULL CHECK (length(content_hash) = 32),
    type            TEXT         NOT NULL CHECK (type IN ('fact', 'decision', 'preference', 'project_context')),
    scope           TEXT         NOT NULL CHECK (scope IN ('global', 'project')),
    project         TEXT,
    source          TEXT         NOT NULL,
    tags            TEXT[]       NOT NULL DEFAULT '{}',
    metadata        JSONB        NOT NULL DEFAULT '{}',
    confidence      NUMERIC(4,3) NOT NULL DEFAULT 1.0 CHECK (confidence > 0.0 AND confidence <= 1.0),
    version         INTEGER      NOT NULL DEFAULT 1 CHECK (version > 0),
    deleted_at      TIMESTAMPTZ,
    deleted_by      TEXT,
    delete_reason   TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    search_vector   TSVECTOR     GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', content), 'A') ||
        setweight(to_tsvector('simple', memora_text_array_search_text(tags)), 'B') ||
        setweight(to_tsvector('simple', source), 'C')
    ) STORED,
    CONSTRAINT memories_project_scope_check CHECK (
        (scope = 'global' AND project IS NULL) OR
        (scope = 'project' AND project IS NOT NULL AND length(project) > 0)
    )
);

CREATE UNIQUE INDEX memories_dedup
    ON memories (scope, coalesce(project, ''), type, content_hash)
    WHERE deleted_at IS NULL;

CREATE INDEX memories_scope_project ON memories (scope, project) WHERE deleted_at IS NULL;
CREATE INDEX memories_type ON memories (type) WHERE deleted_at IS NULL;
CREATE INDEX memories_source ON memories (source) WHERE deleted_at IS NULL;
CREATE INDEX memories_tags ON memories USING GIN (tags) WHERE deleted_at IS NULL;
CREATE INDEX memories_created_at ON memories (created_at DESC);
CREATE INDEX memories_updated_at ON memories (updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX memories_search_vector ON memories USING GIN (search_vector);

CREATE TABLE memory_embeddings (
    memory_id           UUID        NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    profile_id          UUID        NOT NULL REFERENCES embedding_profiles(id),
    embedding           VECTOR      NOT NULL,
    embedding_dimension INTEGER     NOT NULL CHECK (embedding_dimension > 0),
    content_hash        BYTEA       NOT NULL CHECK (length(content_hash) = 32),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (memory_id, profile_id)
);

CREATE INDEX memory_embeddings_profile_id ON memory_embeddings (profile_id);

CREATE TABLE audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    action      TEXT        NOT NULL CHECK (action IN ('remember', 'recall', 'update', 'forget', 'health_check', 'list', 'get_context')),
    memory_id   UUID        REFERENCES memories(id),
    source      TEXT,
    scope       TEXT        CHECK (scope IS NULL OR scope IN ('global', 'project')),
    project     TEXT,
    details     JSONB       NOT NULL DEFAULT '{}',
    duration_ms INTEGER     CHECK (duration_ms IS NULL OR duration_ms >= 0),
    success     BOOLEAN     NOT NULL,
    error_msg   TEXT
);

CREATE INDEX audit_log_occurred_at ON audit_log (occurred_at DESC);
CREATE INDEX audit_log_memory_id ON audit_log (memory_id) WHERE memory_id IS NOT NULL;
CREATE INDEX audit_log_action ON audit_log (action);

CREATE TABLE confirmation_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    memory_id  UUID        NOT NULL REFERENCES memories(id),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX confirmation_tokens_one_unused_per_memory
    ON confirmation_tokens (memory_id) WHERE used_at IS NULL;
CREATE INDEX confirmation_tokens_memory_id ON confirmation_tokens (memory_id);
CREATE INDEX confirmation_tokens_expires_at ON confirmation_tokens (expires_at);

CREATE TABLE reindex_jobs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id      UUID        NOT NULL REFERENCES embedding_profiles(id),
    project         TEXT,
    status          TEXT        NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed')),
    total_count     INTEGER     CHECK (total_count IS NULL OR total_count >= 0),
    processed_count INTEGER     NOT NULL DEFAULT 0 CHECK (processed_count >= 0),
    skipped_count   INTEGER     NOT NULL DEFAULT 0 CHECK (skipped_count >= 0),
    error_count     INTEGER     NOT NULL DEFAULT 0 CHECK (error_count >= 0),
    last_memory_id  UUID,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX reindex_jobs_profile_id ON reindex_jobs (profile_id);
CREATE INDEX reindex_jobs_status ON reindex_jobs (status);
CREATE INDEX reindex_jobs_started_at ON reindex_jobs (started_at DESC);

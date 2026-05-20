# Memora Context Glossary

This file captures canonical product language for Memora. It is a glossary only, not an implementation spec.

## Terms

### Memora
A local-first MCP memory server that lets multiple AI agents on the same machine share persistent memories.

### Memory
An atomic, self-contained fact, decision, preference, or project context item that an agent can save, recall, update, or forget. Memora v1 intentionally keeps the memory type taxonomy small (`fact`, `decision`, `preference`, `project_context`) but the domain model and storage schema should be easy to extend later.

### Agent
An MCP-compatible AI tool or assistant that connects to Memora and uses its memory tools. Initial target agents are Hermes, Claude Code, Codex CLI, and Cursor. The complete v1 MCP tool surface is `remember`, `recall`, `get_context`, `list_memories`, `update_memory`, `forget`, and `health_check`.

### Project
A namespace used to isolate memories that belong to a specific codebase, product, or working context. Memora v1 also supports explicit memory scope: `global` memories are available across projects by default, while `project` memories require a project namespace.

### Source
The agent or process that created or modified a memory. Source is tracked for provenance and auditability, not for ownership isolation. Memora v1 uses source tracking and audit logs but does not implement agent-level ACLs; stdio MCP configuration is the primary access boundary.

### Provenance
Metadata that explains where a memory came from, such as source agent, project, timestamp, and optional session/excerpt details.

### Embedding Profile
A versioned configuration describing how embeddings are produced. An embedding profile includes provider, model, dimension count, distance metric, active/inactive status, and API base URL. Memora v1 must support bring-your-own API keys via environment-variable references and must never store raw API secrets in the database.

### Memory Embedding
A vector representation of one memory under one embedding profile. Memora stores embeddings in a separate `memory_embeddings` table rather than directly on `memories`, so canonical memory content can survive provider/model changes and re-embedding can happen safely.

### Extraction LLM
A configurable chat/completion model used to extract atomic memories from raw text when `auto_extract` is enabled. It is separate from the embedding model.

### Re-embedding
The process of creating or refreshing `memory_embeddings` for an embedding profile. Memora must not silently mix incompatible embedding profiles. In v1, `remember` fails by default if embedding generation fails, so accidental unsearchable memories are not stored.

### Atomic Remember Write
In v1, `remember` validates input, calls the embedding provider first, then opens a short database transaction to insert `memories`, `memory_embeddings`, and `audit_log`. Memora must not hold a database transaction open while waiting on an external embedding API.

### Vector Search Strategy
Memora v1 uses exact pgvector search only on PostgreSQL + pgvector. It does not create HNSW/IVFFlat indexes in v1 because configurable embedding profiles make index lifecycle more complex. Approximate indexes are deferred to v2 after profiling and real usage data.

### Keyword and Hybrid Search
Memora v1 includes simple PostgreSQL full-text keyword search using a generated `tsvector` on canonical memory fields. `recall` supports `semantic`, `keyword`, and `hybrid` modes. v1 hybrid search merges and deduplicates semantic and keyword result sets with simple score boosting; advanced BM25/RRF ranking is deferred to v2.

### Memory Deduplication
Memora v1 always performs exact normalized content hash deduplication within `(scope, project, type)`. It also supports similarity-based dedupe strategies on `remember`: `none`, `warn`, and `skip`; `skip` is the default. Semantic merge is deferred to v2.

### Destructive Operation Safety
Memora v1 uses soft-delete-only `forget` with a two-phase confirmation token flow. `update_memory` requires an exact `memory_id` and expected `version`, increments version, recomputes the active-profile embedding, and records audit logs.

### Encryption at Rest
Memora v1 uses PostgreSQL and does not implement SQLCipher or application-level content encryption. v1 documentation must recommend OS/disk encryption such as LUKS, BitLocker, or FileVault; never store raw API keys; disclose remote embedding privacy implications; and note that soft delete is not secure deletion. Application-level encryption is deferred to v2 research.

### Runtime Dependencies
Memora v1 requires Go 1.26.3 (current stable) for implementation/build and PostgreSQL 16 with pgvector at runtime. The WSL host currently has Go 1.24.3 and must be upgraded to Go 1.26.3 before implementation begins. Memora v1 does not require Redis. The v1 docker-compose file should include Postgres/pgvector only. Redis is deferred to v2 for optional caching, rate limiting, or distributed job coordination if needed.

### Reindex Execution
Memora v1 runs embedding reindexing as a synchronous foreground CLI command, not as a background worker or service. Reindex progress/resume state is stored in PostgreSQL when needed, and unchanged memories are skipped via `content_hash`.

### Health Checks
Memora v1 includes a CLI `memora health` command and a read-only MCP `health_check` tool. It does not expose an HTTP health endpoint in v1. Health checks verify config, DB connectivity, pgvector, migrations, active embedding profile, and embedding provider connectivity.

### CLI Surface
Memora v1 CLI focuses on setup, admin, read-only inspection, export, migration, installation, health, and embedding-profile/reindex operations. Full manual memory CRUD is not included in the v1 CLI; write/delete/update memory operations are exposed through MCP tools.

### Public Go API
Memora v1 includes `pkg/api` as a thin public Go MCP client/types package. It exposes request/response structs and a typed client for the seven MCP tools, with a stdio implementation useful for tests/integrations. It must not expose direct DB access, admin internals, or `internal/` storage types.

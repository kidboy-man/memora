# PRD: Memora v0.1.0

## 1. Product Summary

Memora is a local-first MCP memory server that lets multiple AI agents on the same machine share persistent, searchable knowledge.

Initial target agents:

- Hermes
- Claude Code
- Codex CLI
- Cursor

Memora v0.1.0 provides a production-oriented foundation for agent memory: durable PostgreSQL storage, pgvector semantic search, PostgreSQL full-text keyword search, configurable embedding providers, global/project memory scoping, deduplication, soft-delete safety, audit logging, and a small MCP tool surface designed for local agent workflows.

Memora is local-first for storage and runtime control. If the user configures a remote embedding or extraction provider, memory text is sent to that configured provider. Local embedding providers remain supported for users who prefer stronger privacy and no per-token cost.

## 2. Problem Statement

Local AI agents often operate in isolated contexts. Hermes, Claude Code, Codex CLI, Cursor, and other MCP-compatible agents can each learn facts, project conventions, user preferences, and architectural decisions, but that knowledge is usually trapped inside one tool or one conversation.

The user wants a shared memory layer so any local agent can access the same durable knowledge without repeatedly asking the user to restate preferences, project decisions, environment details, or prior context.

The system must be:

- Local-first and privacy-conscious.
- Production-grade enough to trust for long-lived project memory.
- Designed in Go with clean backend architecture.
- Extensible to future cloud/server mode without compromising v1 simplicity.
- Easy to install and configure for multiple MCP agents.
- Safe around destructive operations, secrets, and provider/API-key handling.

## 3. Goals

1. Provide a local MCP memory server named Memora.
2. Allow multiple MCP-compatible agents to share global and project-scoped memories.
3. Store memories durably in PostgreSQL 16 with pgvector.
4. Support semantic, keyword, and simple hybrid recall.
5. Support configurable embedding providers and bring-your-own API keys.
6. Avoid locking users into one embedding model or vector dimension.
7. Preserve canonical memory content separately from embedding vectors.
8. Support safe model switching through embedding profiles and re-embedding.
9. Expose a small, stable v1 MCP tool surface.
10. Provide CLI setup, health, migration, inspection, export, installation, and embedding-profile administration.
11. Include v1 integration tests using testcontainers-go.
12. Maintain a production Go project structure with clear boundaries between domain, storage, server, CLI, and public API packages.

## 4. Non-Goals for v0.1.0

1. HTTP server mode.
2. Multi-user cloud-hosted deployment.
3. Agent-level ACLs.
4. Redis dependency.
5. HNSW or IVFFlat approximate vector indexes.
6. Advanced BM25/RRF hybrid ranking.
7. Application-level content encryption.
8. Secure physical deletion/purge.
9. Background worker service.
10. Manual full CRUD via CLI.
11. Web UI.
12. Full SDK with direct database/admin APIs.
13. Semantic merge of duplicate memories.

## 5. Versioning, Naming, and Repository

Product name: Memora.

Version target: v0.1.0.

License: Apache-2.0.

Language: Go.

Go target: Go 1.26.3 (current stable as of implementation; minimum Go 1.26).

PostgreSQL target: PostgreSQL 16.

Runtime database extension: pgvector.

Binary name: `memora`.

Go module path: `github.com/kidboy-man/memora`.

Default config directory: `~/.config/memora`.

GitHub repository target: `github.com/kidboy-man/memora`.

## 6. Primary Users and Actors

1. Developer/user
   - Owns the local machine.
   - Configures Memora.
   - Chooses embedding providers and API keys.
   - Reviews, exports, and administers memories.

2. MCP agents
   - Hermes, Claude Code, Codex CLI, Cursor, and future MCP-compatible clients.
   - Save, recall, update, and forget memories using MCP tools.

3. Memora CLI operator
   - Usually the same developer/user.
   - Runs migrations, health checks, setup, installation, reindexing, and exports.

4. Embedding provider
   - OpenRouter, OpenAI-compatible endpoint, Ollama/local, or custom provider.
   - Converts memory/query text into vectors.

5. Extraction LLM provider
   - Optional chat/completion model used only when automatic atomic-memory extraction is enabled.

## 7. User Stories

1. As a developer, I want all local agents to share the same memory, so that I do not have to repeat project facts across tools.
2. As a developer, I want global memories, so that my durable preferences follow me across projects.
3. As a developer, I want project-scoped memories, so that codebase-specific knowledge does not pollute unrelated projects.
4. As a developer, I want Memora to support Hermes, Claude Code, Codex CLI, and Cursor, so that my preferred tools can share context.
5. As an agent, I want to remember an atomic fact, so that future agents can retrieve it.
6. As an agent, I want to recall relevant memories semantically, so that exact wording is not required.
7. As an agent, I want to recall memories by exact keywords, so that file paths, package names, errors, and identifiers are not missed.
8. As an agent, I want hybrid recall, so that semantic and exact-token retrieval both contribute to results.
9. As an agent, I want a project context bundle, so that I can start work with relevant background.
10. As a developer, I want to inspect memories from the CLI, so that I can understand what agents have stored.
11. As a developer, I want to export memories, so that I can back up or inspect my knowledge base.
12. As a developer, I want setup to be interactive, so that I can configure database and embedding provider settings without hand-editing everything.
13. As a developer, I want YAML config with environment-variable references, so that secrets are not hardcoded in config files.
14. As a developer, I want to bring my own OpenRouter API key, so that I can use my existing account and model access.
15. As a developer, I want to use local Ollama embeddings, so that I can avoid per-token cost and remote text disclosure.
16. As a developer, I want embedding dimensions to be auto-detected, so that I do not misconfigure vector storage.
17. As a developer, I want embedding model switching to be explicit, so that old and new vectors are not silently mixed.
18. As a developer, I want re-embedding commands, so that I can migrate memories to a new model.
19. As a developer, I want reindexing to resume/retry, so that interrupted migrations are recoverable.
20. As an agent, I want duplicate memories to be skipped by default, so that the memory store remains clean.
21. As an agent, I want a dedupe warning mode, so that I can see possible duplicates and still insert if necessary.
22. As an agent, I want memory updates to use version checks, so that concurrent updates do not silently overwrite each other.
23. As an agent, I want deletes to require confirmation, so that accidental destructive operations are prevented.
24. As a developer, I want deletes to be soft-deletes in v1, so that accidental deletion can be inspected or recovered.
25. As a developer, I want audit logs, so that I can understand who changed memory and when.
26. As a developer, I want health checks, so that I can diagnose database, migration, pgvector, and provider issues.
27. As an agent, I want a read-only health-check tool, so that I can diagnose Memora availability before relying on it.
28. As a developer, I want Docker Compose for PostgreSQL + pgvector, so that local setup is easy.
29. As a developer, I want no Redis requirement in v1, so that the local stack remains simple.
30. As a developer, I want integration tests against real PostgreSQL/pgvector, so that storage behavior is trustworthy.
31. As a Go developer, I want a thin public Go API client, so that tests and future integrations can call Memora through MCP without depending on internals.
32. As a maintainer, I want clean architecture boundaries, so that storage, embeddings, MCP handlers, and CLI code can evolve independently.
33. As a maintainer, I want structured logging, so that operational failures are diagnosable.
34. As a maintainer, I want secrets scanned before storage, so that accidental credential capture is visible.
35. As a privacy-conscious user, I want documentation to explain remote embedding implications, so that I understand what leaves my machine.

## 8. Functional Requirements

### 8.1 Memory Model

A memory is an atomic, self-contained item that agents can save, recall, update, list, and forget.

v1 memory types:

- `fact`
- `decision`
- `preference`
- `project_context`

The default type is `fact`.

The taxonomy must be easy to extend in future versions, but v1 intentionally keeps it small.

Each memory must include:

- ID
- content
- type
- scope
- project when scope is project
- source agent/process
- tags
- metadata/provenance
- confidence
- content hash
- version
- timestamps
- soft-delete fields

### 8.2 Global and Project Scope

Memora v1 supports explicit memory scope:

- `global`
- `project`

Global memories are visible across projects by default.

Project memories require a project namespace.

Recall for project `X` searches both:

- global memories
- project memories where project equals `X`

`include_global` defaults to true.

Results must include scope and project metadata.

### 8.3 Embedding Profiles

Memora v1 supports configurable embedding profiles.

An embedding profile includes:

- name
- provider
- model
- API base URL
- dimensions
- distance metric
- active/inactive status
- timestamps

API keys must be supplied through environment-variable references, not stored raw in the database.

Default recommended profile:

- provider: OpenRouter
- model: `openai/text-embedding-3-small`
- dimensions: auto-detected, expected 1536
- distance metric: cosine

Alternative supported providers:

- OpenRouter
- OpenAI-compatible endpoints
- Ollama/local
- custom API base URL

Memora must clearly disclose that remote embedding providers receive memory/query text.

### 8.4 Separate Memory Embeddings

Canonical memory content must be stored separately from embedding vectors.

The storage model uses three core concepts:

- memories
- embedding profiles
- memory embeddings

A memory embedding represents one memory under one embedding profile.

This design supports:

- provider switching
- model switching
- dimension changes
- re-embedding
- future multi-profile search
- future index strategies

### 8.5 Remember

`remember` stores a memory and its active-profile embedding.

Behavior:

1. Validate input.
2. Normalize content and compute content hash.
3. Perform exact dedupe.
4. Generate embedding before opening a DB transaction.
5. Perform similarity dedupe if enabled.
6. Open short DB transaction.
7. Insert canonical memory.
8. Insert active-profile memory embedding.
9. Insert audit log.
10. Commit.

Memora must not hold a database transaction open while waiting on an external embedding API.

If embedding generation fails, the memory write fails by default.

If DB write fails after embedding succeeds, the memory is not stored; the wasted embedding call cost is acceptable.

### 8.6 Deduplication

Exact normalized content hash dedupe is always enforced within:

- scope
- project
- type

Similarity dedupe supports strategies:

- `none`
- `warn`
- `skip`

Default strategy: `skip`.

Similarity dedupe checks memories in the same scope/project/type context.

Initial similarity threshold: 0.92 cosine similarity.

`merge` is deferred to v2.

### 8.7 Recall

`recall` retrieves relevant memories.

Inputs include:

- query
- project
- include_global
- mode
- types
- tags
- limit

Modes:

- `semantic`
- `keyword`
- `hybrid`

Default mode: `hybrid`.

Semantic recall uses pgvector exact search over embeddings for the active embedding profile.

Keyword recall uses PostgreSQL full-text search with the `simple` dictionary.

Hybrid recall merges and deduplicates semantic and keyword result sets with simple score boosting when both retrieval modes match.

Advanced BM25/RRF ranking is deferred to v2.

### 8.8 Get Context

`get_context` returns a compact context bundle for a project/task.

It should be optimized for agents at the beginning of a work session.

It may internally call recall/list logic, but its output should be grouped and agent-friendly.

Suggested grouping:

- global preferences
- project decisions
- project context
- relevant facts
- recently updated memories

### 8.9 List Memories

`list_memories` supports browsing/filtering memories.

Inputs include:

- scope
- project
- include_global
- type filter
- tag filter
- source filter
- include_deleted
- pagination cursor/limit

Deleted memories are excluded by default.

### 8.10 Update Memory

`update_memory` updates a memory by exact memory ID.

Requirements:

- exact memory ID required
- expected version required
- optimistic concurrency check required
- version increments on success
- active-profile embedding is recomputed
- audit log is written

### 8.11 Forget

`forget` is soft-delete only in v1.

Requirements:

- exact memory ID required
- two-phase confirmation required
- phase 1 returns preview and confirmation token
- phase 2 requires confirmation token
- deletion records deleted timestamp, deleting source, and reason
- audit log is written

Physical purge is out of scope for v1.

Soft-delete is not secure deletion and documentation must say so.

### 8.12 Health Check

v1 includes:

- CLI `memora health`
- read-only MCP `health_check` tool

No HTTP health endpoint in v1.

Health checks verify:

- config load
- database connectivity
- pgvector extension
- migration status
- active embedding profile exists
- embedding provider connectivity
- memory counts

### 8.13 Re-embedding

Re-embedding creates or refreshes memory embeddings for a profile.

v1 reindexing is a synchronous foreground CLI operation.

It is not a background worker and does not run inside `memora serve`.

Reindex supports:

- profile selection
- optional project scope
- resume/retry
- content-hash skipping of unchanged memories
- progress reporting

`recall` only searches memories with embeddings for the active profile.

`serve` fails if no active embedding profile exists.

`serve` warns but continues if some memories lack active-profile embeddings.

## 9. MCP Tool Surface

v1 exposes exactly seven MCP tools:

1. `remember`
2. `recall`
3. `get_context`
4. `list_memories`
5. `update_memory`
6. `forget`
7. `health_check`

MCP v1 should not expose additional admin tools. Admin operations belong in the CLI unless agents need them later.

Tool annotations should follow current MCP guidance, including read-only/destructive hints where applicable.

Expected annotations:

- `remember`: write operation
- `recall`: read-only
- `get_context`: read-only
- `list_memories`: read-only
- `update_memory`: write operation
- `forget`: destructive operation
- `health_check`: read-only

## 10. CLI Surface

v1 CLI focuses on setup, admin, read-only inspection, export, migration, installation, health, and embedding-profile/reindex operations.

Commands:

```text
memora init
memora serve --stdio
memora migrate up
memora migrate down
memora health

memora install claude-code
memora install codex
memora install cursor
memora install hermes

memora embeddings profiles list
memora embeddings profiles activate <name>
memora embeddings profiles test <name>
memora embeddings reindex --profile <name> [--project <project>] [--resume <job_id>]
memora embeddings verify

memora memories list [--project <project>] [--include-global] [--include-deleted]
memora memories show <memory_id>
memora memories export --format json|jsonl
```

Full manual memory CRUD is not included in the v1 CLI. Write/update/delete memory operations are exposed through MCP tools.

## 11. Configuration

Config format: YAML with environment-variable references.

Default config location: `~/.config/memora/config.yaml`.

`memora init` must interactively create the default config.

Example shape:

```yaml
database:
  host: localhost
  port: 5432
  name: memora
  user: memora
  password: ${MEMORA_DB_PASSWORD}
  sslmode: disable

embedding:
  provider: openrouter
  model: openai/text-embedding-3-small
  api_base_url: https://openrouter.ai/api/v1
  api_key: ${MEMORA_OPENROUTER_API_KEY}
  dimensions: auto
  distance_metric: cosine

extraction_llm:
  provider: openrouter
  model: ${MEMORA_EXTRACTION_MODEL}
  api_base_url: https://openrouter.ai/api/v1
  api_key: ${MEMORA_OPENROUTER_API_KEY}

server:
  name: memora
  version: 0.1.0
  log_level: info

defaults:
  project: default
  include_global: true
  recall_mode: hybrid
  max_results: 10
```

`memora init` should:

1. Ask for database settings.
2. Ask for embedding mode: remote managed or local.
3. Ask for provider/model/API key env var name.
4. Test embedding provider using sample text.
5. Auto-detect dimensions.
6. Create or record initial active embedding profile.
7. Write config file.
8. Provide next-step instructions.

## 12. Storage Requirements

### 12.1 Database

Runtime storage: PostgreSQL 16 with pgvector.

Required extensions:

- pgvector
- pgcrypto or equivalent UUID generation support

Redis is not required in v1.

Docker Compose v1 should include Postgres/pgvector only.

### 12.2 Core Tables

The schema must support these logical tables:

- memories
- embedding_profiles
- memory_embeddings
- audit_log
- confirmation_tokens or signed-token equivalent
- migration bookkeeping
- optional embedding job/progress table for reindex resume

### 12.3 Memories Table

The memories table stores canonical durable memory content and metadata.

Required fields include:

- id
- content
- content_hash
- type
- scope
- project
- source
- tags
- metadata
- confidence
- version
- timestamps
- soft-delete fields
- generated search vector

Validation:

- content must not be empty
- type must be one of v1 memory types
- scope must be global or project
- project is required for project-scoped memories
- confidence must be between 0.0 and 1.0
- version must be positive

Deduplication uniqueness must cover scope, project, type, and content hash.

### 12.4 Embedding Profiles Table

The embedding profiles table stores provider/model/dimension metadata.

Required fields include:

- id
- name
- provider
- model
- API base URL
- dimensions
- distance metric
- active flag
- timestamps

Only one profile should be active by default for recall. Future versions may support multi-profile search.

### 12.5 Memory Embeddings Table

The memory embeddings table stores vectors per memory per embedding profile.

Required fields include:

- memory ID
- profile ID
- embedding vector
- embedding dimension
- content hash
- timestamps

The table must enforce that stored vector dimensions match recorded embedding dimensions.

Primary key should prevent duplicate embeddings for the same memory/profile pair.

### 12.6 Audit Log

Audit log records committed memory operations.

Required fields include:

- id
- timestamp
- action
- memory ID when applicable
- source
- scope/project
- details
- duration
- success flag
- error when applicable

Audit logs are kept forever in v1. Retention policy is deferred to v2.

### 12.7 Row-Level Security

PostgreSQL RLS should be used for project isolation where applicable.

RLS must fail closed.

Project context must be set transaction-locally to avoid leaking state across connections.

Global memory fallback must be explicitly handled by policies or query design.

## 13. Search Requirements

### 13.1 Semantic Search

v1 uses exact pgvector search.

No HNSW or IVFFlat indexes in v1.

Exact search gives perfect recall and avoids index lifecycle complexity while Memora is still local-first and expected to handle relatively small memory volumes.

Initial performance target:

- For up to 10,000 memories per project, recall should complete under 300ms on a typical developer laptop, excluding embedding API latency.

### 13.2 Keyword Search

v1 includes PostgreSQL full-text search.

Use the `simple` dictionary, not English stemming, because memories may contain:

- code identifiers
- file paths
- package names
- model names
- error strings
- acronyms
- mixed-language content

Search vector should include content, tags, and source with appropriate weighting.

### 13.3 Hybrid Search

v1 hybrid search merges semantic and keyword result sets.

It deduplicates by memory ID.

It applies a simple boost when a memory appears in both semantic and keyword results.

Advanced ranking such as BM25/RRF is deferred to v2.

## 14. Embedding Provider Requirements

Memora v1 must support bring-your-own API keys.

Provider support:

- OpenRouter default/recommended remote provider
- OpenAI-compatible direct provider
- Ollama/local provider
- custom base URL provider

Default recommended model:

- `openai/text-embedding-3-small` via OpenRouter

Rationale:

- good quality/cost tradeoff
- expected 1536 dimensions
- compatible with future normal pgvector HNSW indexing
- widely understood and available

Alternative documented options:

- `baai/bge-m3` for multilingual/open-model preference
- `openai/text-embedding-3-large` for higher quality but 3072-dim index caveats
- `mistralai/codestral-embed-2505` for future code-specialized memory profile
- local Ollama models such as `nomic-embed-text`

Implementation must not assume dimensions from config alone. It should test the provider and detect actual vector length during profile creation/testing.

## 15. Extraction LLM Requirements

Extraction LLM is separate from embedding provider.

It is used only when `auto_extract` is enabled.

Purpose:

- convert raw text into atomic memories
- choose one of the allowed v1 memory types
- preserve provenance

Extraction provider should be configurable through the same config style as embedding provider.

If the user supplies already-atomic memory content, extraction LLM is not required.

## 16. Security and Privacy Requirements

### 16.1 API Keys

Memora must never store raw API keys in the database.

Config should use environment-variable references for secrets.

### 16.2 Remote Provider Disclosure

Documentation and init prompts must clearly state that remote embedding/extraction providers receive submitted memory/query text.

### 16.3 Secret Scanning

Before storing memory, Memora should scan content for likely secrets.

Behavior:

- scan and flag, do not block
- return a warning to the caller
- store metadata flag such as `contains_secret: true`

### 16.4 Prompt Injection Mitigation

Memora should sanitize stored content for safe rendering and agent return paths.

Sanitization must not corrupt canonical content, but outputs to agents should be structured so memories are treated as data, not instructions from the system/developer.

### 16.5 Encryption at Rest

v1 does not implement SQLCipher or application-level content encryption.

Documentation must recommend OS/disk encryption:

- LUKS on Linux
- BitLocker on Windows
- FileVault on macOS

Application-level encryption is deferred to v2 research.

### 16.6 Access Control

v1 does not implement agent-level ACLs.

v1 relies on local stdio MCP configuration as the access boundary.

Source tracking and audit logs provide provenance, not authorization.

Agent-level ACLs are deferred to v2, likely alongside HTTP/socket multi-client mode.

## 17. Observability and Operations

### 17.1 Logging

Use structured logging with Zap.

Log key events:

- server startup
- config load
- migration status
- tool calls
- embedding provider failures
- DB failures
- dedupe decisions
- reindex progress
- health-check failures

Do not log raw secrets.

### 17.2 Metrics

Prometheus metrics are deferred to v2.

### 17.3 Health

v1 health is CLI + MCP only.

No HTTP listener in v1.

### 17.4 Audit Retention

Audit logs are kept forever in v1.

Retention policy is deferred to v2.

## 18. Installation and Setup

Supported install paths:

1. Prebuilt binary release.
2. `go install`.
3. Docker image for the binary if useful later.

Recommended first-run flow:

1. Install Memora.
2. Run `memora init`.
3. Start PostgreSQL/pgvector via included Docker Compose if needed.
4. Run migrations.
5. Install MCP config for desired agents.
6. Run `memora serve --stdio` through agent MCP configuration.
7. Run `memora health` to verify.

Docker Compose v1 includes Postgres/pgvector only.

## 19. Go Architecture Requirements

Use a production Go layout with clear separation between executable entrypoints, domain logic, adapters, server handlers, CLI, migrations, tests, and public API.

Architecture style:

- clean/hexagonal architecture
- domain interfaces defined by consumers
- adapters implement domain interfaces
- constructor injection
- no DI framework
- explicit error handling
- context-first functions
- table-driven tests

Major modules:

1. Domain module
   - memory entities
   - embedding profile entities
   - validation
   - dedupe policy
   - ranking/merge logic
   - domain errors

2. Storage module
   - PostgreSQL implementation
   - migrations
   - transaction runner/unit of work
   - RLS context handling

3. Embedding module
   - provider interface
   - OpenRouter/OpenAI-compatible adapter
   - Ollama adapter
   - dimension detection
   - provider health/test calls

4. Extraction module
   - optional LLM extraction interface
   - OpenRouter/OpenAI-compatible implementation
   - atomic memory extraction prompt/validation

5. MCP server module
   - stdio server
   - seven tool handlers
   - middleware for logging/errors/audit where appropriate

6. CLI module
   - init
   - serve
   - migrate
   - install
   - health
   - embeddings administration
   - memory read/export commands

7. Public API module
   - thin Go MCP client
   - request/response types
   - stdio client implementation

8. Test helpers
   - deterministic mock embedder
   - testcontainers PostgreSQL/pgvector setup
   - MCP tool simulation

## 20. Public Go API Scope

`pkg/api` v1 is a thin public Go MCP client/types package.

It exposes:

- request/response structs
- typed client interface for seven MCP tools
- stdio MCP client implementation

It must not expose:

- direct database access
- admin internals
- internal storage types
- migration APIs

The API is v0.x and may evolve before v1.0.0.

## 21. Testing Requirements

### 21.1 Test Layers

v1 includes three test layers:

1. Unit tests
2. Integration tests
3. End-to-end tests

### 21.2 Unit Tests

Unit tests cover:

- domain validation
- memory type/scope validation
- dedupe policy
- hybrid result merge/scoring
- error mapping
- handler behavior with mocked storage/embedder
- config parsing
- secret scanning
- confirmation-token flow

Use table-driven tests for handlers and domain behavior.

Target >80% coverage for domain and handler packages.

### 21.3 Integration Tests

Integration tests use testcontainers-go with real PostgreSQL/pgvector.

They cover:

- migrations
- pgvector exact search
- PostgreSQL full-text search
- hybrid retrieval against real DB
- RLS behavior
- soft-delete filtering
- optimistic update behavior
- audit log writes
- embedding profile persistence

### 21.4 End-to-End Tests

E2E tests simulate an agent through MCP flow:

- remember → recall
- remember duplicate → skip
- update_memory → recall updated content
- forget phase 1 → phase 2 → recall excludes deleted
- health_check returns OK under healthy setup

### 21.5 CI

CI platform: GitHub Actions.

CI should run:

- Go formatting checks
- Go vet/static checks where practical
- unit tests with race detector
- integration tests with PostgreSQL/pgvector
- coverage reporting

Go version in CI: Go 1.26.3.

## 22. Acceptance Criteria

Memora v0.1.0 is acceptable when:

1. `memora init` interactively creates valid config.
2. Docker Compose starts PostgreSQL 16 with pgvector.
3. Migrations create all required tables/extensions/indexes.
4. `memora health` passes on a valid local setup.
5. `memora serve --stdio` serves all seven MCP tools.
6. Claude Code/Codex/Cursor/Hermes install commands generate correct MCP config.
7. `remember` stores memory and embedding atomically.
8. `remember` fails when active-profile embedding generation fails.
9. Exact dedupe skips identical normalized memory content.
10. Similarity dedupe supports `none`, `warn`, and `skip`.
11. `recall` supports semantic, keyword, and hybrid modes.
12. `recall` includes global memories by default for project queries.
13. `get_context` returns grouped, compact context useful to agents.
14. `list_memories` supports filtering and pagination.
15. `update_memory` enforces expected version and recomputes embedding.
16. `forget` uses two-phase soft-delete confirmation.
17. Deleted memories are excluded from normal recall/list results.
18. Audit logs are written for committed memory operations.
19. API keys are not stored raw in the database.
20. Remote provider privacy implications are documented.
21. `embeddings reindex` runs synchronously and can resume/retry.
22. `embeddings verify` reports missing/stale/dimension-mismatched embeddings.
23. `pkg/api` provides typed Go client/types for the seven MCP tools.
24. Unit, integration, and E2E tests pass in CI.
25. Coverage target exceeds 80% for domain and handler code.

## 23. Out of Scope

Out of scope for v0.1.0:

- HTTP API/server mode
- web dashboard
- cloud multi-user deployment
- OAuth or user accounts
- agent-level ACLs
- Redis
- HNSW/IVFFlat index management
- advanced BM25/RRF ranking
- Prometheus metrics
- app-level encrypted memory content
- physical secure deletion
- background workers
- semantic memory merge
- manual full memory CRUD CLI
- mobile app support
- hosted SaaS offering

## 24. Open Questions for Later Versions

1. Should v2 add HTTP or Unix socket multi-client mode?
2. Should v2 add agent-level ACLs and trust tiers?
3. Should v2 add HNSW indexes per active embedding profile?
4. Should v2 support halfvec indexes for 3072-dim embeddings?
5. Should v2 support multi-profile recall across general/code/multilingual embeddings?
6. Should v2 add Redis for cache/rate-limit/job coordination?
7. Should v2 add Prometheus/OpenTelemetry metrics?
8. Should v2 implement application-level encryption?
9. Should v2 add retention policy for audit logs?
10. Should v2 add semantic duplicate merge workflows?
11. Should v2 add a web UI for memory review/editing?
12. Should v2 support hosted/cloud sync?

## 25. Implementation Notes

1. Avoid long database transactions around external API calls.
2. Treat memories returned to agents as data, not instructions.
3. Keep storage and provider interfaces small and testable.
4. Prefer exact correctness in v1 over approximate index performance.
5. Do not leak secrets in logs, audit details, config output, or errors.
6. Keep v1 MCP tool surface small.
7. Keep CLI admin-focused.
8. Keep canonical memory independent of embeddings.
9. Design migrations so v2 index and provider changes can be added without rewriting the memory model.
10. Use explicit, typed domain errors and map them consistently to MCP/tool responses.

## 26. Final Locked v0.1.0 Decisions

- Product/repo/binary/config namespace: Memora / `memora`.
- License: Apache-2.0.
- Language: Go.
- Go target: Go 1.26.3 (minimum Go 1.26).
- Storage: PostgreSQL 16 + pgvector.
- Redis: deferred to v2.
- Transport: MCP stdio v1.
- Runtime mode: local-first.
- Default embedding provider: OpenRouter.
- Default embedding model: `openai/text-embedding-3-small`.
- Embedding dimensions: auto-detected.
- API keys: bring-your-own via env var references.
- Memory schema: canonical memories separate from embeddings.
- Search: exact pgvector + PostgreSQL full-text + simple hybrid.
- ANN indexes: deferred to v2.
- Memory types: fact, decision, preference, project_context.
- Scope: global and project.
- Deduplication: exact always, similarity strategy default skip.
- Forget: soft-delete, two-phase confirmation.
- Update: optimistic versioning.
- Audit logs: included, retained forever in v1.
- Encryption: OS/disk encryption recommended; app-level encryption deferred.
- Health: CLI + MCP tool; no HTTP health endpoint.
- CLI: setup/admin/read/export/reindex only; no full manual CRUD.
- Public API: thin Go MCP client/types package.
- Tests: unit + integration + E2E; testcontainers-go; >80% coverage target.

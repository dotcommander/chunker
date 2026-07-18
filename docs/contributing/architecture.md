# Architecture Overview

Chunker follows **clean architecture** principles with clear separation of concerns across four layers.

## Layer Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Entry Point                              │
│                      cmd/chunker/main.go                         │
│           (Server/CLI mode detection, signal handling)           │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                ┌──────────────┴──────────────┐
                ▼                             ▼
┌───────────────────────────┐     ┌───────────────────────────┐
│      Handler Layer        │     │       CLI Runner          │
│  internal/handler/        │     │    cmd/chunker/cli.go     │
│  - HTTP endpoints         │     │  - Stdin processing       │
│  - Request validation     │     │  - Output formatting      │
└──────────────┬────────────┘     └──────────────┬────────────┘
               │                                │
               └──────────────┬─────────────────┘
                              ▼
              ┌───────────────────────────────┐
              │      Service Layer            │
              │   internal/service/           │
              │  - Orchestration              │
              │  - Validation                 │
              │  - Error handling             │
              └──────────────┬────────────────┘
                             │
                             ▼
              ┌───────────────────────────────┐
              │    Chunking Layer             │
              │  github.com/dotcommander/     │
              │         chunking              │
              │  - Strategy implementations   │
              │  - Token counting             │
              └──────────────┬────────────────┘
                             │
                             ▼
              ┌───────────────────────────────┐
              │      Domain Layer             │
              │    internal/domain/           │
              │  - Core types                 │
              │  - Interfaces                 │
              │  - Business contracts         │
              └───────────────────────────────┘
```

## Data Flow

### Server Mode (HTTP API)

```
Client Request
      │
      ▼
┌─────────────────┐
│ HTTP Handler    │ ← Validates request, extracts parameters
│ /chunk endpoint │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ ChunkService    │ ← Validates business rules, selects strategy
│ .Chunk()        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ chunking.New    │ ← Creates appropriate chunker implementation
│ Chunker/Token   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Strategy        │ ← Executes chunking algorithm (smart_boundary,
│ Implementation  │   token_based, word_boundary, etc.)
│ .Chunk()        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ ChunkResponse   │ ← Returns chunks with metadata
│ + Metadata      │
└────────┬────────┘
         │
         ▼
   JSON Response
```

### CLI Mode (Stdin Processing)

```
Stdin Input
      │
      ▼
┌─────────────────┐
│ CLI Runner      │ ← Detects pipe, reads input, applies flags
│ cli.go          │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ ChunkService    │ ← Same service layer as HTTP mode
│ .Chunk()        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ chunking.New    │ ← Creates chunker based on -strategy flag
│ Chunker/Token   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Strategy        │ ← Same strategy implementations as HTTP mode
│ Implementation  │
│ .Chunk()        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ CLI Formatter   │ ← Outputs JSON/JSONL based on -format flag
│ to stdout       │
└─────────────────┘
```

## Key Interfaces

### Chunker (chunking.Chunker)

The foundational interface for all chunking strategies, defined in the external library.

```go
type Chunker interface {
    // Chunk splits text into pieces based on strategy
    Chunk(ctx context.Context, text string, size int, overlap int) []Chunk
    // Strategy returns the strategy name
    Strategy() Strategy
}
```

**Where to implement:** `github.com/dotcommander/chunking` (external library)

**When to use:** Obtain via `chunking.NewChunker(strategy)` — do not implement locally

### Token counting (chunking.CountTokens)

Standalone function in the external library for token-aware operations.

```go
// CountTokens returns token count for text using the named encoding.
func CountTokens(text, encoding string) (int, error)
```

**Where to use:** Call `chunking.CountTokens(text, encoding)` from service or chunker code.

**When to use:** Computing token counts for LLM context limit checks.

### ChunkService (service.ChunkService)

Orchestrates chunking operations and business logic.

```go
type ChunkService interface {
    // Chunk performs validation and delegates to strategy
    Chunk(ctx context.Context, req ChunkRequest) (*ChunkResponse, error)
}
```

**Where to implement:** `internal/service/chunk_service.go`

**When to modify:** Adding business rules, validation, or orchestration logic

## Layer Responsibilities

### Domain Layer (`internal/domain/`)

**Responsibility:** Core business types and contracts

**Contains:**
- `models.go`: ChunkRequest, ChunkResponse, Chunk, Metadata, Strategy enum
- `interfaces.go`: ChunkService

**Key Rule:** No external dependencies—pure Go types and interfaces

**When to edit:** Changing core types, adding new interfaces

### Service Layer (`internal/service/`)

**Responsibility:** Business logic orchestration

**Contains:**
- `chunk_service.go`: Validates requests, coordinates factory and chunkers

**When to edit:** Adding business rules, cross-cutting concerns

### Chunking Layer (`github.com/dotcommander/chunking`)

**Responsibility:** Strategy implementations (external library)

**Provides:**
- `chunking.NewChunker(strategy)`: Creates a text-based chunker
- `chunking.NewTokenChunker(encoding)`: Creates a token-based chunker
- `chunking.CountTokens(text, encoding)`: Standalone token counting
- Strategy implementations: smart_boundary, sentence_boundary, word_boundary, paragraph_aware, hard_cut, token_based

**When to update:** Bump the dependency version in `go.mod`

### Handler Layer (`internal/handler/`)

**Responsibility:** HTTP request/response handling

**Contains:**
- `chunk_handler.go`: HTTP handlers for `/chunk` and `/health`

**When to edit:** Adding endpoints, changing response format

### Entry Point (`cmd/chunker/`)

**Responsibility:** Application initialization and mode detection

**Contains:**
- `main.go`: Server/CLI detection, signal handling
- `cli.go`: CLI runner with io.Reader/io.Writer injection

**When to edit:** Changing startup behavior, CLI flags

## Design Patterns Used

| Pattern | Location | Purpose |
|---------|----------|---------|
| Strategy | `github.com/dotcommander/chunking` | Pluggable chunking algorithms |
| Dependency Injection | All layers | Testability and decoupling |

## Making Changes

### Adding a New Chunking Strategy

New strategies are implemented in the external `github.com/dotcommander/chunking` library, not in this repo. To consume a new strategy:

1. Bump `github.com/dotcommander/chunking` in `go.mod` / run `go get`
2. Add the strategy constant to `domain/models.go`
3. Update `internal/service/chunk_service.go` to wire `chunking.NewChunker` for the new constant
4. Write integration tests in `internal/service/`

### Modifying Business Logic

- Edit `internal/service/chunk_service.go`
- Add validation to `ChunkRequest.Validate()` in `domain/models.go`

### Adding HTTP Endpoints

- Add handler in `internal/handler/chunk_handler.go`
- Register route in `cmd/chunker/main.go`

### Changing CLI Behavior

- Edit `cmd/chunker/cli.go` for runner logic
- Edit `cmd/chunker/main.go` for flag parsing

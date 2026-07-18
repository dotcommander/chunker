# Architecture

## Overview

Chunker is a high-performance text chunking service and CLI tool written in Go that intelligently splits text into manageable chunks using various strategies (NLP-based, token-counting, word/sentence boundaries). The system follows clean architecture principles with clear separation of concerns across domain, service, and infrastructure layers.

## Core Components

The codebase is organized into distinct layers:

- **Domain Layer** (`internal/domain/`): Core types and interfaces with no external dependencies
- **Service Layer** (`internal/service/`): Business logic orchestration
- **Chunking Delegation** (`github.com/dotcommander/reliquary@v0.7.0`): All strategy implementations live in the external reliquary library; this repo maps types via `internal/domain/convert.go` and orchestrates calls via `internal/service/chunk_service.go`
- **Handler Layer** (`internal/handler/`): HTTP request/response handling
- **Entry Point** (`cmd/chunker/`): CLI and server mode orchestration

## Data Flow

### CLI Mode

1. **Entry Point**: `main.go` detects piped stdin via `os.Stdin.Stat()`
2. **Input Reading**: `cli.go` reads stdin with `io.ReadAll()`
3. **Request Building**: CLI flags mapped to `domain.ChunkRequest`
4. **Service Processing**: `ChunkService.ProcessChunkRequest()` orchestrates chunking
5. **Strategy Execution**: Service calls `chunking.NewChunker` / `chunking.NewTokenChunker` from reliquary
6. **Output Formatting**: JSON or JSONL formatted response to stdout

### Server Mode

1. **HTTP Request**: Chi router receives POST to `/chunk` endpoint
2. **Handler Validation**: `chunk_handler.go` validates JSON request body
3. **Service Processing**: Same `ChunkService` as CLI mode
4. **Strategy Execution**: Service calls `chunking.NewChunker` / `chunking.NewTokenChunker` from reliquary
5. **Response Generation**: JSON response with chunks and metadata
6. **Error Handling**: Typed errors return appropriate HTTP status codes

## Design Patterns

### Strategy Pattern

All chunking algorithms implement the `Chunker` interface:

```go
type Chunker interface {
  Chunk(ctx context.Context, text string, size int, overlap int) []Chunk
  Strategy() Strategy
}
```

**Implementations are delegated to [reliquary](https://github.com/dotcommander/reliquary):**
- **smart_boundary**: abbreviation-aware, punctuation-based sentence detection
- **sentence_boundary**: Simple punctuation-based sentence splitting
- **word_boundary**: Splits at word boundaries respecting chunk size
- **paragraph_aware**: Keeps paragraphs together when possible
- **hard_cut**: Exact character count splits (no boundary respect)
- **token_based**: Token counting for LLM compatibility
- **markdown_aware**: Preserves markdown structure (headings, code blocks, lists)

### Factory Pattern (in reliquary)

Chunker creation is handled by the reliquary library:

```go
// In chunk_service.go:
libChunker, err := chunking.NewChunker(domain.ToLibStrategy(strategy))
// or for token-based:
libChunker, err := chunking.NewTokenChunker(string(req.TokenEncoding.WithDefault()))
```

This repo's `internal/domain/convert.go` maps domain types (`Strategy`, `Chunk`) to/from reliquary types via `ToLibStrategy`, `FromLibChunk`, and `FromLibChunks`. The service layer calls reliquary's constructors directly with no intermediate factory abstraction.

**Benefits:**
- No local factory boilerplate
- Strategy implementations stay in reliquary (single source of truth)
- Adding a strategy in reliquary automatically surfaces it here (after updating domain `Strategy` consts and `IsValid()`))
- Consumer code (`chunk_service.go`) remains unchanged when new strategies are added

### Dependency Injection

All services receive dependencies through constructor injection:

```go
func NewChunkService() *ChunkService {
  return &ChunkService{}
}

func NewChunkHandler(service ChunkService) *ChunkHandler {
  return &ChunkHandler{
    chunkService: service,
  }
}
```

**Benefits:**
- Testable (can inject mocks)
- Flexible (different configurations for different contexts)
- No global state or hidden dependencies

### Interface Segregation

Minimal, focused interfaces:

- `Chunker`: 2 methods (Chunk, Strategy) — provided by reliquary
- `TokenChunker`: Extends Chunker for token-specific operations — provided by reliquary
- `ChunkService`: 1 method (ProcessChunkRequest)

All interfaces <5 methods, following Interface Segregation Principle.

## Key Design Decisions

### Decision 1: Sentence-Level Overlap vs Character-Level

**Context**: Overlapping chunks preserve context across boundaries

**Decision**: smart_boundary uses sentence-level overlap; word_boundary uses character-level

**Rationale**:
- Sentence-level preserves semantic units (no mid-sentence fragmentation)
- Character-level is faster (no re-parsing)
- Different strategies optimize for different use cases

**Consequences**:
- ✅ Semantic preservation for NLP/LLM use cases
- ❌ O(n²) re-parsing overhead for smart_boundary
- ⚠️  Users must understand strategy-specific overlap semantics

### Decision 2: Token Counting vs Character Counting

**Context**: LLM context windows measured in tokens, not characters

**Decision**: token_based chunking counts word/punctuation tokens, not characters

**Rationale**:
- Accurate token estimation prevents LLM context overruns
- Simple `strings.FieldsFunc()` tokenization for predictable performance
- Matches tiktoken-go library token counting behavior

**Consequences**:
- ✅ Accurate LLM context window management
- ❌ Token count ≠ character count (varies by text density)
- ⚠️  Users must understand token vs character distinction

### Decision 3: Pure Domain Models with No External Dependencies

**Context**: Domain package uses plain Go for validation, with no external validation library.

**Decision**: Validation lives in `domain.ChunkRequest.Validate()` — a plain Go method that returns errors.

**Rationale**:
- Dependency Inversion Principle (DIP) compliance
- Domain models should have zero external dependencies
- Easier testing and reasoning

**Consequences**:
- ✅ 100% DIP compliance (no package-level globals)
- ✅ Domain models are pure value objects
- ✅ No external validation library dependency

### Decision 4: Context-Aware Cancellation

**Context**: Long-running chunking operations should be cancellable

**Decision**: All `Chunk()` methods accept `context.Context` as first parameter

**Rationale**:
- Enables HTTP request timeout handling
- Graceful shutdown when clients disconnect
- Standard Go idiom for cancellation

**Consequences**:
- ✅ Operations can be cancelled mid-processing
- ✅ Resource cleanup on timeout
- ✅ Consistent with Go standard library patterns

## Database Schema

N/A - Chunker is stateless with no persistence layer.

## Integration Points

### External Libraries

**Reliquary (`github.com/dotcommander/reliquary`):**
- Purpose: All chunking strategy implementations (smart_boundary, token_based, word_boundary, sentence_boundary, hard_cut, paragraph_aware, markdown_aware)
- Usage: Called via `chunking.NewChunker` / `chunking.NewTokenChunker` from `internal/service/chunk_service.go`
- Type mapping: `internal/domain/convert.go` maps between domain and library types

**Tiktoken-Go (`github.com/pkoukk/tiktoken-go`):**
- Purpose: Token encoding for TokenBased strategy
- Encodings: cl100k_base, o200k_base, p50k_base, r50k_base
- Usage: Accurate token counting matching OpenAI LLM tokenizers

**Chi Router (`github.com/go-chi/chi/v5`):**
- Purpose: HTTP routing and middleware
- Middleware: Logger, Recoverer, RequestID, RealIP, Timeout (60s)
- Usage: Server mode HTTP API

### API Consumers

**HTTP API:**
- Base URL: `http://localhost:8080`
- Primary endpoint: `POST /chunk`
- Health check: `GET /health`
- Authentication: None (intended for local/internal use)

**CLI Piping:**
- Input: stdin (pipe text through)
- Output: stdout (JSON or JSONL)
- Error output: stderr
- Usage: `cat file.txt | chunker -size 1000 -strategy smart_boundary`

## Component Dependencies

```
main.go
├─> cli.go (CLI mode)
│   ├─> ChunkService
│   │   └─> reliquary (chunking.NewChunker / chunking.NewTokenChunker)
│   └─> Output formatters (JSON/JSONL)
└─> server setup (Server mode)
    └─> ChunkHandler
        └─> ChunkService
            └─> reliquary (chunking.NewChunker / chunking.NewTokenChunker)
```

**Dependency Rules:**
- Domain types have no dependencies (pure Go)
- Service layer depends on domain interfaces
- Handlers depend on service interfaces
- Main depends on handlers and services
- Chunking strategies are delegated to reliquary (external library)

**No circular dependencies** - clean architecture enforced.

## Request Lifecycle

### CLI Request

1. User pipes text: `echo "text" | chunker -size 1000`
2. `main.go` detects piped stdin
3. `cli.go` reads stdin, builds `ChunkRequest`
4. `ChunkService` validates request
5. Service calls `chunking.NewChunker` / `chunking.NewTokenChunker` from reliquary
6. Strategy chunker processes text
7. Response formatted as JSON/JSONL
8. Output written to stdout

### HTTP Request

1. Client POSTs to `/chunk` with JSON body
2. Chi router routes to `ChunkHandler`
3. Handler validates Content-Type and JSON
4. `ChunkService.ProcessChunkRequest()` called
5. Validation checks (empty text, invalid size, overlap >= size)
6. Service calls `chunking.NewChunker` / `chunking.NewTokenChunker` from reliquary
7. `Chunker.Chunk()` processes text with context timeout
8. Response serialized to JSON
9. HTTP 200 OK or 400 Bad Request returned

## Core Business Logic

### Validation Rules

**Request validation** (`domain.ChunkRequest.Validate()`):
- Text must not be empty
- ChunkSize must be > 0
- Overlap must be >= 0
- Overlap must be < ChunkSize

**Strategy validation**:
- Strategy must be one of: smart_boundary, word_boundary, sentence_boundary, hard_cut, paragraph_aware, token_based, markdown_aware
- If empty, defaults to smart_boundary

**Token encoding validation** (TokenBased strategy):
- Encoding must be one of: cl100k_base, o200k_base, p50k_base, r50k_base
- If empty, defaults to cl100k_base

### Chunking Algorithm Characteristics

All strategies are implemented in reliquary. The descriptions below summarize behavior as exposed through this service.

**smart_boundary:**
- Abbreviation-aware, punctuation-based sentence detection
- Handles abbreviations (Dr., Inc.), decimals (3.14), complex punctuation
- Sentence-level overlap (re-parses previous chunk)
- O(n) first pass, O(n²) with overlap
- Best for: Documents with complex sentences, technical writing

**word_boundary:**
- Pre-tokenizes with `strings.Fields()`
- Concatenates words until size limit
- Character-level overlap
- O(n) time complexity
- Best for: High-throughput, simple tokenization acceptable

**sentence_boundary:**
- Simple regex/split on sentence endings (`.!?`)
- Two-level hierarchy: paragraphs → sentences
- Respects paragraph boundaries (`\n\n`)
- O(n) time complexity
- Best for: Formatted documents with clear paragraph structure

**hard_cut:**
- Exact character count splits
- Configurable overlap with stride calculation
- No boundary respect (may split mid-word)
- O(n) time complexity
- Best for: Fixed-size requirements, predictable memory usage

**paragraph_aware:**
- Prioritizes keeping paragraphs together
- Falls back to sentence splitting if paragraph too large
- O(n) time complexity
- Best for: Multi-paragraph documents where paragraph is semantic unit

**token_based:**
- Tokenizes on whitespace and punctuation
- Counts tokens (not characters)
- Token count varies by text density (words + punctuation)
- O(n) time complexity
- Best for: LLM context window management, accurate token estimation

**markdown_aware:**
- Preserves markdown structure (headings, code blocks, lists)
- O(n) time complexity
- Best for: Markdown documents where structural units are meaningful

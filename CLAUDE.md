# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Chunker is a high-performance text chunking service written in Go that splits text into manageable chunks using various strategies (NLP-based, token-based, word/sentence boundaries). It operates in two modes: HTTP API server and CLI tool with stdin processing.

## Development Commands

### Building and Running

```bash
# Build the binary
make build

# Build and run immediately
make run

# Run server mode (default 127.0.0.1:8080)
./bin/chunker serve

# Expose the server on all interfaces with a custom port
./bin/chunker serve -bind 0.0.0.0 -port 3000

# CLI mode (pipe text through stdin)
echo "Your text here" | ./bin/chunker -size 1000 -strategy smart_boundary
cat document.txt | ./bin/chunker -format jsonl
```

### Testing

```bash
# Run all tests
make test
go test -v ./...

# Run tests for a specific package
go test -v ./internal/service/

# Run a specific test
go test -v ./internal/service/ -run TestChunkService_ProcessChunkRequest_Success
go test -v ./cmd/chunker/ -run TestCLIRunner

# Run tests with coverage
go test -cover ./...
```

### Linting and Code Quality

```bash
# Run linter (requires golangci-lint)
make lint

# Download/update dependencies
make deps
go mod tidy
```

### Cross-Platform Builds

```bash
# Build for multiple platforms
make build-all  # Creates darwin-amd64, linux-amd64, windows-amd64 binaries
```

## Architecture

The project follows **clean architecture** with clear separation of concerns:

### Layer Separation

**Domain Layer** (`internal/domain/`)
- `models.go`: Core types (ChunkRequest, ChunkResponse, Chunk, Metadata, Strategy, TokenEncoding)
- `interfaces.go`: ChunkService boundary consumed by the HTTP handler and CLI
- **No external dependencies** - pure business logic and contracts

**Service Layer** (`internal/service/`)
- `chunk_service.go`: Orchestrates chunking through reliquary constructors and chunkers
- Implements ChunkService interface
- Handles validation and error responses

**Chunking Integration (delegated to `github.com/dotcommander/reliquary`)**
- `internal/service/chunk_service.go`: orchestrates chunking; calls `chunking.NewChunker` / `chunking.NewTokenChunker`, converts results, computes metadata
- `internal/domain/convert.go`: maps between domain and library types (`ToLibStrategy`, `FromLibChunk`, `FromLibChunks`)
- Strategy implementations live in reliquary, not in this repo

**Handler Layer** (`internal/handler/`)
- `chunk_handler.go`: HTTP handlers for `/chunk` and `/health` endpoints
- Validates requests via `domain.ChunkRequest.Validate()` (plain Go)
- Returns JSON responses with proper error handling

**Entry Point** (`cmd/chunker/`)
- `main.go`: Dual-mode detection (server vs CLI based on stdin pipe detection)
- `cli.go`: CLI runner implementation with io.Reader/io.Writer dependency injection
- `cli_test.go`: CLI tests with table-driven approach

### Key Design Patterns

**Strategy Delegation**: Different chunking algorithms implement reliquary's `chunking.Chunker` interface
**Library Delegation**: ChunkService selects reliquary constructors based on the requested strategy
**Dependency Injection**: Handler and CLI components receive the chunk service and I/O dependencies through constructors

## Dependencies

### Core
- `github.com/go-chi/chi/v5`: HTTP router with middleware support
- `github.com/dotcommander/reliquary`: chunking strategies (smart_boundary, token_based, etc.)
- `github.com/pkoukk/tiktoken-go`: Transitive reliquary dependency for token encoding (cl100k_base, o200k_base, p50k_base, r50k_base)

### Testing
- Standard library `testing` package
- Table-driven tests for chunking strategies

## Key Concepts

### Chunking Strategies

The system supports seven strategies, each delegated to [reliquary](https://github.com/dotcommander/reliquary):

1. **smart_boundary** (default): abbreviation-aware sentence detection (handles "Dr. Smith", "U.S.A.", etc.)
2. **sentence_boundary**: Basic sentence splitting using punctuation (`.!?`)
3. **word_boundary**: Splits at word boundaries, never breaks words
4. **paragraph_aware**: Prioritizes keeping paragraphs together (splits on `\n\n`)
5. **hard_cut**: Exact character count, may split mid-word
6. **token_based**: Counts tokens using tiktoken encodings for LLM context limits
7. **markdown_aware**: Preserves markdown structure (headings, code blocks, lists)

### Overlap Behavior

All strategies support overlap between chunks for context preservation:
- Overlap is measured in characters (or tokens for token_based strategy)
- Previous chunk's tail overlaps with next chunk's head
- Validation ensures overlap < chunk_size

### Token Encodings

For `token_based` strategy, supports multiple encodings:
- `cl100k_base`: GPT-3.5/GPT-4 (default)
- `o200k_base`: GPT-4o, GPT-5 models
- `p50k_base`: Older models (GPT-3 Codex)
- `r50k_base`: Legacy models (GPT-2)

### Mode Detection

The binary automatically determines operation mode:
- **Server mode**: Explicitly requested with the `serve` subcommand
- **CLI mode**: Stdin is piped (detected via `os.Stdin.Stat()`)
- Shows help if neither condition is met

## Common Patterns

### Adding a New Chunking Strategy

Strategies are implemented in reliquary. To surface one here:
1. Add the `Strategy` const to `internal/domain/models.go`
2. Update `IsValid()` to include the new strategy
3. Map it in `internal/domain/convert.go` if needed
4. Ensure reliquary's `NewChunker` handles the strategy

### Testing Chunking Strategies

Use table-driven tests with test cases covering:
- Basic chunking (text fits in one chunk)
- Multiple chunks with overlap
- Edge cases (empty text, single word, very long words)
- Boundary conditions (text exactly at chunk size)

Example structure:
```go
tests := []struct {
    name      string
    input     string
    chunkSize int
    overlap   int
    want      int // expected number of chunks
}{
    {"empty text", "", 100, 0, 0},
    {"single chunk", "short", 100, 0, 1},
    // ...
}
```

### Error Handling

- Request validation uses `domain.ChunkRequest.Validate()` (plain Go)
- Custom validation in `ChunkRequest.Validate()` for business rules
- Service returns errors for unknown strategies
- HTTP handlers return actionable 400 responses for validation, 408 for canceled requests, and generic 500 responses for internal failures
- Service layer propagates errors with context

## Server Configuration

The server binds to `127.0.0.1:8080` by default. Configure persistent defaults
with `server_bind` and `server_port` in `~/.config/chunker/config.yaml`, or use
`chunker serve -bind <address> -port <port>` for a single run. Set `-bind 0.0.0.0`
only when the service should be reachable from other hosts.

External exposure should sit behind a reverse proxy that provides authentication
and transport security; Chunker's HTTP server does not authenticate requests.

### Middleware Stack (chi)
1. Logger: Request logging
2. Recoverer: Panic recovery
3. RequestID: Unique request tracking
4. RealIP: Client IP extraction
5. Timeout: 60-second request timeout

### Graceful Shutdown
- Listens for SIGINT/SIGTERM signals
- 30-second shutdown timeout for in-flight requests
- Clean server stop with connection draining

## CLI Usage Patterns

The CLI expects text via stdin and outputs JSON:

```bash
# Basic usage
cat file.txt | chunker

# Custom chunk size and strategy
echo "text" | chunker -size 500 -strategy word_boundary -overlap 50

# Token-based chunking for LLM processing
cat code.py | chunker -size 2000 -strategy token_based -encoding cl100k_base

# JSON Lines output for streaming
cat book.txt | chunker -format jsonl | while read line; do echo "$line" | jq .; done

# Pretty-printed JSON
echo "text" | chunker -pretty
```

## API Response Structure

All `/chunk` responses include:
- `chunks[]`: Array of chunk objects with id, text, char_count, word_count, token_count (if applicable)
- `metadata`: Summary with total_chunks, total_chars, total_tokens (if applicable), strategy_used, token_encoding (if applicable)

See API.md for complete endpoint documentation and integration examples.

## Code Style Conventions

- Use dependency injection (pass dependencies to constructors)
- Return errors, don't panic (except in main initialization)
- Context as first parameter for all chunking operations
- Table-driven tests with descriptive test case names
- Interface-first design (define contracts before implementations)
- Keep domain layer free of external dependencies

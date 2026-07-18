# Adding a New Chunking Strategy

## Overview

Chunking strategies are implemented in **[github.com/dotcommander/reliquary](https://github.com/dotcommander/reliquary)**, not in this repository. Reliquary owns the `chunking.Chunker` interface, concrete strategy implementations, the `Strategy` string constants, and the `chunking.NewChunker` / `chunking.NewTokenChunker` constructors.

**This repository** (`github.com/dotcommander/chunker`) surfaces existing reliquary strategies through the HTTP API and CLI. It owns:

- **`internal/domain/models.go`** — domain `Strategy` consts, `IsValid()`, and `WithDefault()`.
- **`internal/domain/convert.go`** — `ToLibStrategy` (a direct string cast to `chunking.Strategy`), plus `FromLibChunk` / `FromLibChunks` projections to the wire DTO.
- **`internal/service/chunk_service.go`** — `ProcessChunkRequest`, which calls `chunking.NewChunker` (or `chunking.NewTokenChunker` for the token-based strategy), then converts results.

Current strategies: `word_boundary`, `sentence_boundary`, `hard_cut`, `paragraph_aware`, `token_based`, `smart_boundary`, `markdown_aware`.

## Adding a Strategy

### Case A: the strategy already exists in reliquary

1. **Add the Strategy const to `internal/domain/models.go`.**
   The const string value **must exactly match** reliquary's `chunking.Strategy` value. `ToLibStrategy` is a direct string cast, so any mismatch means the factory in reliquary won't recognise it.

   ```go
   const (
       // … existing strategies …
       MyNewStrategy Strategy = "my_new_strategy"
   )
   ```

2. **Add the value to `IsValid()` in `internal/domain/models.go`.**

   ```go
   func (s Strategy) IsValid() bool {
       switch s {
       case WordBoundary, SentenceBoundary, HardCut, ParagraphAware, TokenBased, SmartBoundary, MarkdownAware, MyNewStrategy:
           return true
       }
       return false
   }
   ```

3. **Check `convert.go` and `chunk_service.go` for special handling.**
   `ToLibStrategy` needs no change unless the strategy requires a non-standard code path in `ProcessChunkRequest` (as `token_based` does — it routes through `chunking.NewTokenChunker` instead of `chunking.NewChunker`). If the strategy is a plain chunker, no change is needed.

4. **Add or refresh documentation.**
   Create a strategy doc under `docs/strategies/` following existing conventions. Update the strategy summary in `docs/README.md`.

### Case B: the strategy does not exist in reliquary

Implement it in reliquary first:

1. Open a PR in [`dotcommander/reliquary`](https://github.com/dotcommander/reliquary).
2. Implement the `chunking.Chunker` interface: `Chunk(text string, size int, overlap int) []Chunk` and `Strategy() Strategy`.
3. Add the `Strategy` const in reliquary and a case in `chunking.NewChunker`.
4. Merge the reliquary PR, then follow **Case A** above.

## Testing

Strategy behaviour is tested where it lives — in **reliquary**.

In this repo, add or update:

- **Service-level tests** in `internal/service/chunk_service_test.go` (`TestChunkService_ProcessChunkRequest_*`) when the mapping or integration path warrants coverage.
- **Doc-tests** for strategy docs under `docs/strategies/*_doc_test.go`.

Run the scoped test suite:

```bash
go test ./internal/service/
```

## Checklist

- [ ] Strategy const added to `internal/domain/models.go` with a value matching reliquary's `chunking.Strategy`
- [ ] Value added to the `IsValid()` switch in `internal/domain/models.go`
- [ ] No unintended special handling in `internal/domain/convert.go` or `internal/service/chunk_service.go` (or special handling added if required, e.g. a `chunking.NewTokenChunker` path)
- [ ] Strategy doc added or updated under `docs/strategies/`
- [ ] `go build ./...` passes
- [ ] `go test ./internal/service/` passes

## Reference

Example: how `word_boundary` is declared and wired.

`internal/domain/models.go`:

```go
const (
    WordBoundary Strategy = "word_boundary"
    // …
)
```

`internal/domain/convert.go`:

```go
func ToLibStrategy(s Strategy) chunking.Strategy {
    return chunking.Strategy(s) // direct cast — values must match
}
```

`internal/service/chunk_service.go`:

```go
libChunker, err = chunking.NewChunker(domain.ToLibStrategy(strategy))
```

The constant string `"word_boundary"` flows through `ToLibStrategy` to `chunking.NewChunker`, which dispatches to the implementation registered in reliquary. No local strategy file or factory case is required in this repo.

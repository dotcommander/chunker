# Chunker

A high-performance text chunking service and CLI tool written in Go that intelligently splits text into manageable chunks using various strategies including NLP-based sentence detection and token counting.

## Features

- **Dual Mode Operation**: Run as HTTP API server or CLI tool
- **Smart Chunking Strategies**:
  - `smart_boundary` (default): delegated to `github.com/dotcommander/reliquary` (abbreviation-aware, punctuation-based sentence detection)
  - `word_boundary`: Splits at word boundaries  
  - `sentence_boundary`: Splits at sentence endings
  - `hard_cut`: Exact character count splits
  - `paragraph_aware`: Keeps paragraphs together
  - `markdown_aware`: Preserves markdown headings and fenced code blocks
  - `token_based`: Token counting for LLM compatibility
- **Inspect Mode**: Preview chunking stats and sample chunks without full payload output
- **Batch Files Mode**: Chunk many files from a glob pattern into an output directory
- **Token Encoding Support**: cl100k_base, o200k_base, p50k_base, r50k_base
- **Configurable overlap** between chunks for context preservation
- **Multiple output formats**: JSON, JSON Lines (JSONL)
- **Zero dependencies** for basic operation

## Installation

### From source
```bash
git clone https://github.com/dotcommander/chunker
cd chunker
just build

# Optional: Install globally
sudo cp bin/chunker /usr/local/bin/
# Or symlink to your Go bin
ln -s $(pwd)/bin/chunker ~/go/bin/chunker
```

### Using go install
```bash
go install github.com/dotcommander/chunker/cmd/chunker@latest
```

## Usage

### CLI Mode

Chunker automatically detects when text is piped to stdin:

```bash
# Basic usage with smart defaults
cat document.txt | chunker

# Token-based chunking for LLM processing
cat article.md | chunker -size 2000 -strategy token_based -encoding cl100k_base

# Output as JSON Lines for streaming
cat book.txt | chunker -format jsonl | while read line; do
  echo "$line" | jq .
done

# Custom settings with pretty output
echo "Your text here..." | chunker -size 500 -overlap 50 -pretty

# Show help
chunker

# Inspect chunking quality quickly (NDJSON by default)
cat document.txt | chunker inspect -sample 3

# Human-readable inspect output
cat document.txt | chunker inspect -inspect-format human -sample 3
```

### Server Mode

Run as an HTTP API server:

```bash
# Start server on default port 8080
chunker serve

# Custom port
chunker serve -port 3000

# Or use environment variable
PORT=3000 chunker serve
```

### Files Mode

Chunk multiple files by glob and write output files:

```bash
# Write one output file per input file (JSON by default)
chunker files -out-dir chunks "documents/*.md"

# Write JSONL output files
chunker files -format jsonl -out-dir chunks "documents/*.txt"
```

## CLI Options

```
-size       Chunk size in characters or tokens (default: 4000)
-strategy   Chunking strategy (default: smart_boundary)
            Options: smart_boundary, word_boundary, sentence_boundary,
                    hard_cut, paragraph_aware, markdown_aware, token_based
-overlap    Overlap between chunks (default: 200)
-encoding   Token encoding for token_based strategy (default: cl100k_base)
            Options: cl100k_base, o200k_base, p50k_base, r50k_base
-format     Output format: json, jsonl (default: json)
-pretty     Pretty print JSON output

Server mode:
chunker serve [-port 8080]

Inspect mode:
chunker inspect [-sample 2] [-inspect-format ndjson|human]

Files mode:
chunker files [-out-dir chunks] "<glob>"
```

## Task Runner Migration

This project now uses `just` instead of Task.

```bash
# Before
task build
task test
task lint
task server

# Now
just build
just test
just lint
just server
```

## API Reference

See [API.md](API.md) for complete HTTP API documentation.

### Quick Example

```bash
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Your text here...",
    "chunk_size": 1000,
    "strategy": "smart_boundary",
    "overlap": 100
  }'
```

## Examples

### Pipeline with LLM Tools

```bash
# Chunk and process with an LLM tool
cat research_paper.pdf | pdftotext -layout - | chunker -size 3000 -strategy smart_boundary | jq -r '.chunks[].text' | llm prompt "Summarize this section"

# Process code files with token limits
find . -name "*.go" -exec cat {} \; | chunker -size 4000 -strategy token_based -encoding cl100k_base -format jsonl
```

### Batch Processing

```bash
#!/bin/bash
# Process multiple files
for file in documents/*.txt; do
  cat "$file" | chunker -size 2000 | jq -c '.chunks[]' > "chunks/$(basename "$file" .txt).jsonl"
done
```

## Architecture

The project follows clean architecture principles:

```
chunker/
├── cmd/chunker/        # CLI and server entry point
├── internal/
│   ├── domain/         # Core types and interfaces
│   ├── service/        # Business logic
│   └── handler/        # HTTP handlers
├── pkg/chunking/       # Chunking strategies
└── API.md             # API documentation
```

## Performance

- Processes large texts efficiently with streaming support
- Memory usage scales linearly with text size
- All strategies run in O(n) time complexity
- Concurrent request handling in server mode

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see LICENSE file for details

# CLI Reference

Command-line interface for the Chunker text chunking service.

## Overview

The Chunker CLI supports piped chunking plus explicit subcommands:

- **CLI Mode**: Activates automatically when stdin is piped (e.g., `cat file.txt | chunker`)
- **Help Mode**: `chunker` with no stdin prints help
- **Server Mode**: `chunker serve`
- **Inspect Mode**: `chunker inspect` (reads stdin and prints NDJSON summary)
- **Files Mode**: `chunker files` (chunks files matched by glob)

## CLI Flags

All CLI flags are optional. Smart defaults are provided for optimal text chunking.

| Flag        | Type   | Default | Description                                                                 |
|-------------|--------|---------|-----------------------------------------------------------------------------|
| `-size`     | `int`  | `4000`  | Chunk size in characters (or tokens for `token_based` strategy)             |
| `-strategy` | `string` | `smart_boundary` | Chunking strategy to apply                                           |
| `-overlap`  | `int`  | `200`   | Number of characters/tokens to overlap between consecutive chunks           |
| `-encoding` | `string` | `cl100k_base` | Token encoding (only for `token_based` strategy)                      |
| `-format`   | `string` | `json`  | Output format: `json` or `jsonl`                                            |
| `-pretty`   | `bool` | `false` | Pretty-print JSON output with indentation                                   |
| `-h`        | `bool` | `false` | Display help text                                                           |

### Flag Details

#### `-size`

Specifies the maximum size for each chunk.

- **Character-based strategies** (`smart_boundary`, `sentence_boundary`, `word_boundary`, `paragraph_aware`, `markdown_aware`, `hard_cut`): Size is measured in characters
- **Token-based strategy** (`token_based`): Size is measured in tokens using the specified encoding

#### `-strategy`

Determines the chunking algorithm. See [Strategies](#strategies) for details.

| Value               | Description                                                      |
|---------------------|------------------------------------------------------------------|
| `smart_boundary`    | NLP-based sentence detection (handles abbreviations, etc.)      |
| `sentence_boundary` | Basic sentence splitting using punctuation                       |
| `word_boundary`     | Splits at word boundaries, never breaks words                    |
| `paragraph_aware`   | Prioritizes keeping paragraphs together                          |
| `markdown_aware`    | Preserves markdown headings and fenced code blocks               |
| `hard_cut`          | Exact character count, may split mid-word                        |
| `token_based`       | Counts tokens using specified encoding for LLM context limits    |

#### `-overlap`

Specifies overlap between consecutive chunks for context preservation.

- Overlap is taken from the end of the previous chunk and added to the start of the next chunk
- Must be less than `chunk_size`
- Measured in characters for character-based strategies, tokens for `token_based`

#### `-encoding`

Token encoding for the `token_based` strategy. Ignored for other strategies.

| Value          | Description                           |
|----------------|---------------------------------------|
| `cl100k_base`  | GPT-3.5, GPT-4, GPT-4o (default)      |
| `o200k_base`   | GPT-4o, GPT-5 models                  |
| `p50k_base`    | Older models (GPT-3 Codex)            |
| `r50k_base`    | Legacy models (GPT-2)                 |

#### `-format`

Controls output format.

| Format | Description                                              |
|--------|----------------------------------------------------------|
| `json` | Single JSON object with `chunks` array and `metadata`    |
| `jsonl`| JSON Lines format - one JSON object per line             |

See [Output Formats](#output-formats) for detailed examples.

#### `-pretty`

Enables human-readable JSON output with 2-space indentation.

- Applies to both `json` and `jsonl` formats
- Useful for debugging and human inspection
- Omit for production use (smaller output size)

## Stdin Piping Patterns

The CLI mode requires text input via stdin. Here are common piping patterns:

### Basic File Piping

```bash
# Pipe a file through chunker
cat document.txt | chunker

# Using input redirection
chunker < document.txt

# Process multiple files
cat file1.txt file2.txt | chunker
```

### Command Output Piping

```bash
# Chunk command output
ls -la | chunker
grep "pattern" logfile.txt | chunker
curl -s https://example.com | chunker
```

### Pipeline Integration

```bash
# Chain with other text processing tools
cat document.txt | chunker -size 500 | jq '.chunks[].text'
cat document.txt | chunker -format jsonl | grep '"type": "chunk"'
```

### Output Redirection

```bash
# Save to file
cat input.txt | chunker > output.json
cat input.txt | chunker -format jsonl > output.jsonl

# Combine with other tools
cat input.txt | chunker | jq '.metadata.total_chunks'
```

## Output Formats

### JSON Format (`-format json`)

Default output format. Returns a single JSON object containing all chunks and metadata.

**Example:**

```json
{
  "chunks": [
    {
      "id": 0,
      "start_char": 0,
      "end_char": 45,
      "text": "The quick brown fox jumps over the lazy dog.",
      "char_count": 45,
      "word_count": 9
    },
    {
      "id": 1,
      "start_char": 45,
      "end_char": 73,
      "text": "The lazy dog was not amused.",
      "char_count": 28,
      "word_count": 6
    }
  ],
  "metadata": {
    "total_chunks": 2,
    "total_chars": 73,
    "strategy_used": "smart_boundary"
  }
}
```

**Use cases:**
- Processing with JSON tools (`jq`, `python`, etc.)
- Storing complete results in a single file
- API integration scenarios

### JSONL Format (`-format jsonl`)

JSON Lines format. One JSON object per line for streaming processing.

**Structure:**
- First line: Metadata object with `type: "metadata"`
- Subsequent lines: Individual chunk objects with `type: "chunk"`

**Example:**

```json
{"type":"metadata","metadata":{"total_chunks":2,"total_chars":73,"strategy_used":"smart_boundary"}}
{"type":"chunk","chunk":{"id":0,"start_char":0,"end_char":45,"text":"The quick brown fox jumps over the lazy dog.","char_count":45,"word_count":9}}
{"type":"chunk","chunk":{"id":1,"start_char":45,"end_char":73,"text":"The lazy dog was not amused.","char_count":28,"word_count":6}}
```

**Use cases:**
- Streaming large datasets without loading into memory
- Line-by-line processing with Unix tools
- Real-time data pipelines
- Log processing

**Processing JSONL output:**

```bash
# Count chunks
cat document.txt | chunker -format jsonl | grep -c '"type": "chunk"'

# Extract only chunk text
cat document.txt | chunker -format jsonl | grep '"type": "chunk"' | jq -r '.chunk.text'

# Process line by line
cat document.txt | chunker -format jsonl | while read -r line; do
  echo "$line" | jq '.'
done
```

## Examples

### Basic Usage

```bash
# Use all defaults
cat article.txt | chunker

# Pretty-print for readability
cat article.txt | chunker -pretty
```

### Token-Based Chunking for LLMs

```bash
# Chunk for GPT-4 with 2000 tokens per chunk
cat code.py | chunker -size 2000 -strategy token_based -encoding cl100k_base

# Chunk for GPT-4o with o200k_base encoding
cat document.txt | chunker -size 4000 -strategy token_based -encoding o200k_base
```

### Custom Overlap

```bash
# Increase overlap for more context preservation
cat transcript.txt | chunker -size 1000 -overlap 200

# No overlap between chunks
cat document.txt | chunker -size 500 -overlap 0
```

### Strategy Selection

```bash
# Keep paragraphs together
cat essay.txt | chunker -strategy paragraph_aware

# Word boundary splitting for code
cat script.sh | chunker -strategy word_boundary -size 200

# Exact character count (may break words)
cat data.txt | chunker -strategy hard_cut -size 100

# Preserve markdown blocks
cat README.md | chunker -strategy markdown_aware -size 800
```

### Inspect Mode

Inspect-specific flags:

- `-sample` number of chunk previews to emit (default: `2`)
- `-inspect-format` output format: `ndjson` (default) or `human`

```bash
# Summarize chunking result with NDJSON records (default)
cat document.txt | chunker inspect

# Show more previews
cat document.txt | chunker inspect -sample 5

# Human-readable inspect output
cat document.txt | chunker inspect -inspect-format human
```

### Files Mode

```bash
# Chunk markdown files and write JSON outputs
chunker files -out-dir chunks "docs/*.md"

# Chunk text files and write JSONL outputs
chunker files -format jsonl -out-dir chunks "data/*.txt"
```

### Streaming Processing

```bash
# Stream chunks as JSONL
cat large_book.txt | chunker -format jsonl | while read -r line; do
  type=$(echo "$line" | jq -r '.type')
  if [ "$type" = "chunk" ]; then
    echo "$line" | jq -r '.chunk.text' | process_chunk.py
  fi
done
```

### Combining with Other Tools

```bash
# Extract chunk count with jq
cat document.txt | chunker | jq '.metadata.total_chunks'

# Get total character count
cat document.txt | chunker | jq '.metadata.total_chars'

# Extract first chunk text
cat document.txt | chunker | jq '.chunks[0].text'

# Get all chunk texts as array
cat document.txt | chunker | jq '[.chunks[].text]'
```

## Server Mode

Use the `serve` subcommand to run Chunker as an HTTP API server.

```bash
# Start server on loopback using the configured port (127.0.0.1:8080 by default)
chunker serve

# Expose on all interfaces with a custom port
chunker serve -bind 0.0.0.0 -port 3000

# Override port with environment variable
PORT=5000 chunker serve
```

Server mode provides `/chunk` and `/health` endpoints. See [HTTP API documentation](./schemas.md) for details.
For network exposure, place Chunker behind an authenticating TLS reverse proxy;
the built-in HTTP server does not authenticate requests.

## Error Handling

The CLI exits with a non-zero status code on errors and prints error messages to stderr.

```bash
# Missing input
chunker
# Shows help text

# Invalid strategy
cat file.txt | chunker -strategy invalid
# Error: error chunking text: invalid strategy "invalid"
```

## Environment Variables

| Variable | Description                                  |
|----------|----------------------------------------------|
| `PORT`    | Server port (server mode only, default: 8080) |

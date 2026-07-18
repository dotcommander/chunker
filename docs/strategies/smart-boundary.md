# Smart Boundary Strategy

## Overview

The `smart_boundary` strategy uses abbreviation-aware, punctuation-based sentence detection
delegated to [github.com/dotcommander/reliquary](https://github.com/dotcommander/reliquary) to intelligently identify
sentence boundaries. This is the default strategy and recommended for most
natural language text processing.

## How It Works

### Abbreviation-Aware Sentence Detection

Smart boundary scans for sentence terminators (`.`, `!`, `?`) and suppresses
breaks after known abbreviations (Mr., Dr., etc.) and decimal points.
Fenced and indented code blocks are treated as atomic units.
Unlike simple punctuation-based splitting, this approach:

- Handles edge cases like abbreviations
- Respects quoted sentences
- Maintains context across punctuation

### Abbreviation Handling

The strategy correctly handles common abbreviations that contain periods:

```text
Dr. Smith went to Washington.
Mr. Jones lives in the U.S.A.
The version is 3.14.5.
```

Each of these is recognized as a single sentence, not split at the periods.

### Chunking Process

1. **Sentence Detection**: Text is segmented into sentences using abbreviation-aware detection
2. **Sentence Accumulation**: Sentences are added to a chunk until the
   character count would exceed `size`
3. **Overlap Management**: The last N characters from each chunk are kept
   for the next chunk (when `overlap > 0`)
4. **Chunk Creation**: When full, the chunk is finalized and a new chunk
   starts

## Fallback Behavior

When sentence segmentation yields no usable spans (e.g. malformed input),
a fallback splitter is used — the strategy automatically falls back to
`sentence_boundary` chunking. This ensures robustness even with:

- Malformed Unicode text
- Symbol-only input (e.g., `123 @!# ###`)
- Empty or whitespace-only content
- Text that yields zero sentences

## Usage Examples

### CLI

```bash
# Default strategy (smart_boundary)
cat document.txt | chunker -size 1000

# Explicit specification
cat document.txt | chunker -size 1000 -strategy smart_boundary
```

### API

```bash
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Dr. Smith arrived. He was happy.",
    "chunk_size": 1000,
    "strategy": "smart_boundary"
  }'
```

## Overlap Behavior

Overlap is calculated in characters. The strategy keeps the last N characters
from the current chunk and prepends them to the next chunk:

```go
text := "First sentence. Second sentence. Third sentence."
chunkSize := 40
overlap := 15

// Chunk 0: "First sentence. Second sentence."
// Chunk 1: "Second sentence. Third sentence."
//           ^^^^^^^^^^^^^^^-- 15 char overlap
```

## Edge Cases

| Input | Behavior |
|-------|----------|
| Empty string | Returns empty chunk array |
| Size <= 0 | Returns empty chunk array |
| Single long sentence | Creates one chunk with full sentence |
| Text without sentence-ending punctuation | Treated as single sentence |
| Only symbols/numbers | Falls back to sentence_boundary |

## Performance Considerations

- Abbreviation-aware detection is slower than simple regex splitting
- Recommended for natural language text under 100KB
- For large documents or high-throughput scenarios, consider
  `sentence_boundary` for better performance

## When to Use

| Scenario | Recommendation |
|----------|----------------|
| Natural language text | Use smart_boundary |
| Contains many abbreviations | Use smart_boundary |
| High-performance requirement | Use sentence_boundary |
| Code/technical content | Use hard_cut or token_based |
| Preserving paragraphs | Use paragraph_aware |

## Comparison to sentence_boundary

| Feature | smart_boundary | sentence_boundary |
|---------|----------------|-------------------|
| Abbreviation handling | Yes (Dr., Mr., etc.) | No |
| Performance | Slower | Faster |
| Dependencies | reliquary | None |
| Accuracy | Higher | Lower |
| Fallback | Falls back to sentence_boundary | N/A |

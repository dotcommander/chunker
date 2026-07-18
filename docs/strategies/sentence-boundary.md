# Sentence Boundary Strategy

## Overview

The `sentence_boundary` strategy splits text at sentence boundaries using
the same abbreviation-aware, punctuation-based sentence detection as
`smart_boundary`, delegated to
[github.com/dotcommander/reliquary](https://github.com/dotcommander/reliquary).
It differs from `smart_boundary` only in how overlap content is selected and
how sentences are joined into chunks, providing a simpler overlap model.

## How It Works

### Abbreviation-Aware Sentence Detection

Both `sentence_boundary` and `smart_boundary` share the same sentence
segmentation engine (`splitIntoSentencesWithRuneSpans`) in reliquary.
This engine:

- Splits on sentence terminators (`.`, `!`, `?`)
- Suppresses breaks after known abbreviations (Mr., Dr., Prof., etc.)
- Handles decimal points (avoids splitting "3.14" into "3" and "14")
- Treats fenced and indented code blocks as atomic units

The two strategies produce **identical** sentence segments for any given
input. There is no accuracy difference between them.

### Abbreviation Handling

The strategy correctly handles common abbreviations that contain periods:

```text
Dr. Smith went to Washington.
Mr. Jones lives in the U.S.A.
The version is 3.14.5.
```

Each of these is recognized as a single sentence, not split at the periods.

### Chunking Process

1. **Sentence Detection**: Text is segmented into sentences using reliquary's
   abbreviation-aware detection
2. **Sentence Accumulation**: Sentences are added to a chunk until the
   rune count would exceed `size`
3. **Overlap Management**: The last N sentences from each chunk are kept
   for the next chunk (when `overlap > 0`)
4. **Chunk Creation**: When full, the chunk is finalized and a new chunk
   starts

### Difference from smart_boundary

The two strategies differ only in how they build chunks from the same set of
detected sentences:

| Aspect | sentence_boundary | smart_boundary |
|--------|-------------------|----------------|
| **Sentence detection** | Same (`splitIntoSentencesWithRuneSpans`) | Same |
| **Overlap selection** | Keeps all added sentences for overlap | Tail-counted by byte budget |
| **Sentence joining** | Preserves trailing spaces, appends `" "` only when needed | Joins with `" "` separator |
| **Overlap write** | Writes overlap sentences joined by space | Writes overlap text up to byte budget |

In practice, `sentence_boundary` tracks all sentences added to the current
chunk and uses them as the overlap buffer for the next chunk (sentence-level
overlap). `smart_boundary` selects trailing sentences whose combined byte
length fits within the overlap budget.

## Usage Examples

### CLI

```bash
cat document.txt | chunker -size 1000 -strategy sentence_boundary
```

### API

```bash
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "First sentence. Second sentence. Third sentence.",
    "chunk_size": 50,
    "strategy": "sentence_boundary"
  }'
```

## Example Behavior

### Basic Splitting

```text
Input: "The quick brown fox. He jumped over the fence. Then he ran away."
Chunk size: 40 characters

Result:
Chunk 0: "The quick brown fox. He jumped over the"
Chunk 1: "fence. Then he ran away."
```

### With Overlap

```text
Input: "First sentence. Second sentence. Third sentence."
Chunk size: 30 characters
Overlap: 10 characters

Result:
Chunk 0: "First sentence. Second sentence."
Chunk 1: "Second sentence. Third sentence."
          ^^^^^^^^^^^^^^^-- overlap from end of Chunk 0
```

## Fallback Behavior

When punctuation-based sentence detection produces a single oversized segment
(e.g., bullet lists, YAML, logs, or code without sentence-ending punctuation),
the strategy falls back to splitting on double-newlines, then on single
newlines.

## Edge Cases

| Input | Behavior |
|-------|----------|
| Empty string | Returns empty chunk array |
| Size <= 0 | Returns empty chunk array |
| Single long sentence (> size) | Emitted as standalone chunk |
| Text without sentence-ending punctuation | Falls back to newline splitting |
| Only symbols/numbers | Falls back to newline splitting |

## Overlap Behavior

Overlap is measured in **characters**. The strategy keeps the N most recently
added sentences and prepends them to the next chunk. Overlap content is
applied at sentence boundaries, preserving sentence integrity.

## Performance Considerations

- Pure-Go punctuation scan with abbreviation lookups — no external NLP
- Shares the same segmentation engine as `smart_boundary`
- Suitable for both small and large documents
- No accuracy trade-off vs `smart_boundary`; the difference is in overlap
  semantics

## When to Use

| Scenario | Recommendation |
|----------|----------------|
| Simple sentence-level overlap tracking | ✅ Use sentence_boundary |
| Byte-budgeted overlap needed | Use smart_boundary |
| Natural language text | Either (same accuracy) |
| Contains "Dr.", "Mr.", "U.S.A.", etc. | Either (both handle abbreviations) |
| Code/technical content | Use hard_cut or token_based |
| Preserving paragraphs | Use paragraph_aware |
| Large documents (>100KB) | Either (same performance characteristics) |

## Comparison to smart_boundary

| Feature | sentence_boundary | smart_boundary |
|---------|-------------------|----------------|
| Sentence detection | Same (reliquary) | Same (reliquary) |
| Abbreviation handling | Yes | Yes |
| Decimal-point handling | Yes | Yes |
| Code-block atomicity | Yes | Yes |
| Accuracy | Identical | Identical |
| Overlap model | Sentence-level | Byte-budgeted |
| Dependencies | reliquary | reliquary |

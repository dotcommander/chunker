# Word Boundary Strategy

## Overview

The `word_boundary` strategy splits text at word boundaries, guaranteeing that
words are never broken mid-word. This strategy is ideal for preserving word
integrity in natural language processing, search indexing, and content display
scenarios.

## Core Guarantee

**Never breaks mid-word** - The strategy ensures that each chunk contains only
complete words. If a word would cross the chunk boundary, the entire word is
moved to the next chunk.

## How It Works

### Word Splitting Algorithm

1. **Tokenization**: Text is split into word units using whitespace boundaries
2. **Word Accumulation**: Words are added to a chunk until adding another word
   would exceed `size`
3. **Boundary Detection**: When the chunk size limit is reached, the current
   word is pushed to the next chunk
4. **Overlap Management**: The last N characters from each chunk are kept for
   the next chunk (when `overlap > 0`)

### Long Word Handling

If a single word is longer than `chunk_size`, the strategy includes the entire
word in a single chunk rather than breaking it. This preserves the word boundary
guarantee at the cost of potentially exceeding the target chunk size.

```go
// Example: Very long word
text := "The antidisestablishmentarianism movement"
chunkSize := 20

// Result: Chunk 0 contains the entire 28-character word
// "antidisestablishmentarianism" even though it exceeds chunkSize=20
```

## Usage Examples

### CLI

```bash
# Word-safe chunking
cat document.txt | chunker -size 1000 -strategy word_boundary

# With overlap for context preservation
cat document.txt | chunker -size 500 -strategy word_boundary -overlap 50
```

### API

```bash
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "The quick brown fox jumps over the lazy dog",
    "chunk_size": 20,
    "strategy": "word_boundary"
  }'
```

## Examples

### Basic Word-Safe Splits

```text
Input: "The quick brown fox jumps over the lazy dog"
Size:  20 characters

Chunk 0: "The quick brown" (15 chars)
Chunk 1: "fox jumps over" (15 chars)
Chunk 2: "the lazy dog" (12 chars)
```

Note that "brown" and "fox" are split across chunks, but no word is broken.

### With Overlap

```text
Input:  "The quick brown fox jumps over the lazy dog"
Size:   25 characters
Overlap: 10 characters

Chunk 0: "The quick brown fox" (19 chars)
Chunk 1: "fox jumps over the" (18 chars)
          ^^^-- 10 char overlap from previous chunk
Chunk 2: "the lazy dog" (12 chars)
```

### Long Words

```text
Input:  "Supercalifragilisticexpialidocious is a word"
Size:   20 characters

Chunk 0: "Supercalifragilisticexpialidocious" (34 chars)
         ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^-- Exceeds size but kept intact
Chunk 1: "is a word" (9 chars)
```

## Overlap Behavior

Overlap is calculated in characters and applied at the word level:

1. When a chunk is finalized, the strategy extracts trailing words
2. Words are added to the overlap buffer until the character limit is reached
3. The overlap buffer is prepended to the next chunk

```go
text := "First second third fourth fifth"
chunkSize := 20
overlap := 10

// Chunk 0: "First second third"
// Overlap buffer: ["second", "third"] (17 chars -> trimmed to 10)
// Chunk 1: "second third fourth"
//            ^^^^^^^^^^^-- 10 char overlap
```

## Edge Cases

| Input | Behavior |
|-------|----------|
| Empty string | Returns empty chunk array |
| Size <= 0 | Returns empty chunk array |
| Single word | Creates one chunk with the word |
| Word longer than size | Word kept intact in single chunk |
| Consecutive whitespace | Treated as single delimiter |
| Leading/trailing whitespace | Trimmed from chunks |

## Performance Considerations

- Word boundary splitting is faster than NLP-based strategies
- Memory efficient: processes text in a single pass
- Scales linearly with text length O(n)
- No external dependencies

## When to Use

| Scenario | Recommendation |
|----------|----------------|
| Preserving word integrity | Use word_boundary |
| Search indexing | Use word_boundary |
| Display/UI text | Use word_boundary |
| Sentence context needed | Use smart_boundary or sentence_boundary |
| Code/technical content | Use hard_cut or token_based |
| Token counting required | Use token_based |

## Comparison to Other Strategies

| Feature | word_boundary | smart_boundary | hard_cut |
|---------|---------------|----------------|----------|
| Breaks mid-word | Never | Never | Yes |
| Performance | Fast | Slower | Fastest |
| Sentence awareness | No | Yes | No |
| Long word handling | Keeps intact | Keeps intact | Breaks |
| Best for | Word integrity | Natural language | Exact sizes |

## Technical Details

### Word Tokenization

The strategy uses `strings.Fields()` for word splitting, which splits on
whitespace and treats consecutive whitespace as a single delimiter. Punctuation
remains attached to words (e.g., "word." is a single token).

### Whitespace Preservation

Original whitespace between words is preserved within chunks. Only leading and
trailing whitespace is trimmed from each chunk using `strings.TrimSpace()`.

### Chunk Size Calculation

The chunk size is calculated on the accumulated text including spaces between
words. The size limit is checked before adding each word to ensure the boundary
is respected.

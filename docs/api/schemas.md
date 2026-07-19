# API Schemas

Request and response schemas for the Chunker API.

## ChunkRequest

The request payload for the `/chunk` endpoint.

| Field          | Type           | Required | Default         | Description                                                           |
|----------------|----------------|----------|-----------------|-----------------------------------------------------------------------|
| `text`         | `string`       | Yes      | -               | The input text to be chunked                                          |
| `chunk_size`   | `int`          | Yes      | -               | Maximum size (in characters or tokens) for each chunk                 |
| `strategy`     | `string`       | No       | `smart_boundary` | Chunking strategy to apply                                            |
| `overlap`      | `int`          | No       | `0`             | Number of characters/tokens to overlap between consecutive chunks     |
| `token_encoding` | `string`     | No       | `cl100k_base`   | Token encoding to use (only for `token_based` strategy)               |

### Validation Rules

- `text` must be non-empty
- `chunk_size` must be greater than `0`
- `overlap` must be greater than or equal to `0`
- `overlap` must be strictly less than `chunk_size`
- `strategy` must be one of the valid strategy values (see below)
- `token_encoding` must be one of the valid encoding values (see below)

### Strategy Values

| Value               | Description                                                      |
|---------------------|------------------------------------------------------------------|
| `smart_boundary`    | NLP-based sentence detection (handles abbreviations, etc.)      |
| `sentence_boundary` | Basic sentence splitting using punctuation                       |
| `word_boundary`     | Splits at word boundaries, never breaks words                    |
| `paragraph_aware`   | Prioritizes keeping paragraphs together                          |
| `markdown_aware`    | Preserves markdown headings and fenced code blocks               |
| `hard_cut`          | Exact character count, may split mid-word                        |
| `token_based`       | Counts tokens using specified encoding for LLM context limits    |

### Token Encoding Values

| Value          | Description                           |
|----------------|---------------------------------------|
| `cl100k_base`  | GPT-3.5, GPT-4, GPT-4o (default)      |
| `o200k_base`   | GPT-4o, GPT-5 models                  |
| `p50k_base`    | Older models (GPT-3 Codex)            |
| `r50k_base`    | Legacy models (GPT-2)                 |

### Example Request

```json
{
  "text": "The quick brown fox jumps over the lazy dog. The dog was not amused.",
  "chunk_size": 50,
  "strategy": "smart_boundary",
  "overlap": 10,
  "token_encoding": "cl100k_base"
}
```

## ChunkResponse

The response payload from the `/chunk` endpoint.

| Field     | Type               | Required | Description                           |
|-----------|--------------------|----------|---------------------------------------|
| `chunks`  | `array[Chunk]`     | Yes      | Array of chunk objects                |
| `metadata`| `Metadata`         | Yes      | Summary information about the chunking operation |

### Example Response

```json
{
  "chunks": [
    {
      "id": 0,
      "start_char": 0,
      "end_char": 45,
      "text": "The quick brown fox jumps over the lazy dog.",
      "char_count": 45,
      "word_count": 9,
      "token_count": 13
    },
    {
      "id": 1,
      "start_char": 45,
      "end_char": 69,
      "text": "The dog was not amused.",
      "char_count": 24,
      "word_count": 5,
      "token_count": 7
    }
  ],
  "metadata": {
    "total_chunks": 2,
    "total_chars": 69,
    "total_tokens": 20,
    "strategy_used": "smart_boundary",
    "token_encoding": "cl100k_base"
  }
}
```

## Chunk

Represents a single text chunk.

| Field         | Type     | Required | Description                                          |
|---------------|----------|----------|------------------------------------------------------|
| `id`          | `int`    | Yes      | Zero-based index of the chunk in the sequence        |
| `start_char`  | `int`    | Yes      | Start offset in original text (rune index)           |
| `end_char`    | `int`    | Yes      | End offset in original text (rune index, exclusive)  |
| `text`        | `string` | Yes      | The chunk text content                               |
| `char_count`  | `int`    | Yes      | Number of characters in the chunk                    |
| `word_count`  | `int`    | Yes      | Number of words in the chunk                         |
| `token_count` | `int`    | No       | Number of tokens (only present for `token_based`)    |

## Metadata

Summary information about the chunking operation.

| Field            | Type     | Required | Description                                        |
|------------------|----------|----------|----------------------------------------------------|
| `total_chunks`   | `int`    | Yes      | Total number of chunks generated                   |
| `total_chars`    | `int`    | Yes      | Total character count across all chunks            |
| `total_tokens`   | `int`    | No       | Total token count (only for `token_based` strategy)|
| `strategy_used`  | `string` | Yes      | The strategy applied to generate chunks            |
| `token_encoding` | `string` | No       | Token encoding used (only for `token_based`)       |

## Error Response

Error responses follow a standard JSON format.

| Field    | Type     | Required | Description                        |
|----------|----------|----------|------------------------------------|
| `error`  | `string` | Yes      | Human-readable error message       |

### Example Error Response

```json
{
  "error": "text is required"
}
```

## HTTP Status Codes

| Code | Description                             |
|------|-----------------------------------------|
| `200`| Success                                 |
| `400`| Bad Request (validation error)          |
| `405`| Method Not Allowed (non-POST request)   |
| `408`| Request Timeout (canceled or expired)   |
| `500`| Internal Server Error                   |

# Chunker API Documentation

## Base URL
```
http://localhost:8080
```

## Endpoints

### POST /chunk

Splits text into chunks based on the specified strategy and size limit.

#### Request

**Headers:**
```
Content-Type: application/json
```

**Body:**
```json
{
  "text": "string (required) - The text to be chunked",
  "chunk_size": "integer (required) - Maximum size of each chunk (characters or tokens based on strategy)",
  "strategy": "string (optional) - Chunking strategy to use",
  "overlap": "integer (optional) - Number of characters/tokens to overlap between chunks",
  "token_encoding": "string (optional) - Token encoding for token-based chunking"
}
```

**Strategies:**
- `smart_boundary` (default) - Advanced sentence detection using NLP
- `word_boundary` - Splits at word boundaries, respecting chunk_size
- `sentence_boundary` - Splits at sentence endings when possible
- `hard_cut` - Exact character count splits
- `paragraph_aware` - Tries to keep paragraphs together
- `token_based` - Splits by token count (requires token_encoding)

**Token Encodings** (for token_based strategy):
- `cl100k_base` (default) - Used by GPT-3.5-turbo, GPT-4
- `o200k_base` - Used by GPT-4o models
- `p50k_base` - Used by older models
- `r50k_base` - Used by very old models

#### Response

**Success (200 OK):**
```json
{
  "chunks": [
    {
      "id": 0,
      "text": "chunk content",
      "char_count": 998,
      "word_count": 150,
      "token_count": 250
    }
  ],
  "metadata": {
    "total_chunks": 5,
    "total_chars": 4500,
    "total_tokens": 1250,
    "strategy_used": "smart_boundary",
    "token_encoding": "cl100k_base"
  }
}
```

**Error (400 Bad Request):**
```json
{
  "error": "error message"
}
```

### GET /health

Health check endpoint.

#### Response

**Success (200 OK):**
```json
{
  "status": "ok"
}
```

## Usage Examples

### 1. Basic Word Boundary Chunking

Split a simple text into 50-character chunks:

```bash
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "The quick brown fox jumps over the lazy dog. This is a simple test of the chunking service.",
    "chunk_size": 50
  }'
```

**Response:**
```json
{
  "chunks": [
    {
      "id": 0,
      "text": "The quick brown fox jumps over the lazy dog.",
      "char_count": 45,
      "word_count": 9
    },
    {
      "id": 1,
      "text": "This is a simple test of the chunking service.",
      "char_count": 47,
      "word_count": 9
    }
  ],
  "metadata": {
    "total_chunks": 2,
    "total_chars": 92,
    "strategy_used": "word_boundary"
  }
}
```

### 2. Sentence Boundary with Overlap

Split text at sentence boundaries with 20-character overlap:

```bash
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "First sentence here. Second sentence is longer. Third sentence completes the paragraph. Fourth sentence starts new context.",
    "chunk_size": 80,
    "strategy": "sentence_boundary",
    "overlap": 20
  }'
```

### 3. Hard Cut for Fixed-Size Chunks

Create exact 100-character chunks:

```bash
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris.",
    "chunk_size": 100,
    "strategy": "hard_cut"
  }'
```

### 4. Paragraph-Aware Chunking

Keep paragraphs together when possible:

```bash
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "First paragraph contains introductory content.\n\nSecond paragraph has more detailed information about the topic.\n\nThird paragraph concludes the discussion.",
    "chunk_size": 100,
    "strategy": "paragraph_aware"
  }'
```

### 5. Large Text Processing

Process a large document with overlap for context preservation:

```bash
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d @- << 'EOF'
{
  "text": "Chapter 1: Introduction\n\nIn the beginning, there was a need for better text processing. This led to the development of sophisticated chunking algorithms.\n\nChapter 2: Implementation\n\nThe implementation phase involved careful consideration of various strategies. Each strategy was designed to handle specific use cases.\n\nChapter 3: Results\n\nThe results showed significant improvements in text processing efficiency.",
  "chunk_size": 150,
  "strategy": "paragraph_aware",
  "overlap": 30
}
EOF
```

### 6. Handling Long Words

Process text with words longer than chunk size:

```bash
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "The word supercalifragilisticexpialidocious is quite long. Other words are normal.",
    "chunk_size": 20,
    "strategy": "word_boundary"
  }'
```

### 7. Empty Text Handling

```bash
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "",
    "chunk_size": 100
  }'
```

**Response:**
```json
{
  "chunks": [],
  "metadata": {
    "total_chunks": 0,
    "total_chars": 0,
    "strategy_used": "word_boundary"
  }
}
```

### 8. Smart Boundary Chunking (Default)

Using advanced NLP for better sentence detection:

```bash
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Dr. Smith went to the U.S.A. yesterday. He bought 3.14 kg of apples! What do you think? The meeting is at 5 p.m. tomorrow.",
    "chunk_size": 50,
    "strategy": "smart_boundary"
  }'
```

### 9. Token-Based Chunking

Chunk by token count using specific encodings:

```bash
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Your long text content here that needs to be chunked for processing...",
    "chunk_size": 1000,
    "strategy": "token_based",
    "token_encoding": "cl100k_base",
    "overlap": 100
  }'
```

### 10. Different Token Encodings

Use different encodings for various use cases:

```bash
# For newer models (GPT-4o)
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Technical documentation...",
    "chunk_size": 2000,
    "strategy": "token_based",
    "token_encoding": "o200k_base"
  }'

# For older models
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Legacy content...",
    "chunk_size": 1500,
    "strategy": "token_based",
    "token_encoding": "p50k_base"
  }'
```

### 11. Error Cases

#### Missing required field:
```bash
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Some text"
  }'
```

**Response (400 Bad Request):**
```json
{
  "error": "chunk_size must be positive"
}
```

#### Invalid strategy:
```bash
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Some text",
    "chunk_size": 50,
    "strategy": "invalid_strategy"
  }'
```

**Response (400 Bad Request):**
```json
{
  "error": "unknown strategy: invalid_strategy"
}
```

#### Overlap larger than chunk size:
```bash
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Some text",
    "chunk_size": 50,
    "overlap": 60
  }'
```

**Response (400 Bad Request):**
```json
{
  "error": "overlap must be less than chunk_size"
}
```

## Testing with Files

You can also chunk content from files:

```bash
# Using a file
curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d "{
    \"text\": \"$(cat document.txt)\",
    \"chunk_size\": 1000,
    \"strategy\": \"paragraph_aware\"
  }"

# Using jq for better formatting
cat document.txt | jq -Rs '{text: ., chunk_size: 1000}' | \
  curl -X POST http://localhost:8080/chunk \
  -H "Content-Type: application/json" \
  -d @-
```

## Performance Considerations

- The service handles texts of any size efficiently
- Memory usage scales linearly with text size
- All strategies run in O(n) time complexity
- Overlap processing adds minimal overhead
- The service can handle concurrent requests

## Integration Examples

### Python
```python
import requests
import json

def chunk_text(text, chunk_size=1000, strategy="smart_boundary", overlap=0, token_encoding=None):
    payload = {
        "text": text,
        "chunk_size": chunk_size,
        "strategy": strategy,
        "overlap": overlap
    }
    if token_encoding:
        payload["token_encoding"] = token_encoding
    
    response = requests.post("http://localhost:8080/chunk", json=payload)
    return response.json()

# Example usage - character-based
result = chunk_text("Your long text here...", chunk_size=500)
for chunk in result["chunks"]:
    print(f"Chunk {chunk['id']}: {chunk['word_count']} words")

# Example usage - token-based
result = chunk_text(
    "Your long text here...", 
    chunk_size=1000, 
    strategy="token_based",
    token_encoding="cl100k_base"
)
for chunk in result["chunks"]:
    print(f"Chunk {chunk['id']}: {chunk['token_count']} tokens")
```

### JavaScript/Node.js
```javascript
const axios = require('axios');

async function chunkText(text, chunkSize = 1000, strategy = 'word_boundary', overlap = 0) {
    const response = await axios.post('http://localhost:8080/chunk', {
        text,
        chunk_size: chunkSize,
        strategy,
        overlap
    });
    return response.data;
}

// Example usage
chunkText('Your long text here...', 500)
    .then(result => {
        result.chunks.forEach(chunk => {
            console.log(`Chunk ${chunk.id}: ${chunk.word_count} words`);
        });
    });
```

### Go
```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
)

type ChunkRequest struct {
    Text      string `json:"text"`
    ChunkSize int    `json:"chunk_size"`
    Strategy  string `json:"strategy,omitempty"`
    Overlap   int    `json:"overlap,omitempty"`
}

func chunkText(text string, chunkSize int) (*ChunkResponse, error) {
    req := ChunkRequest{
        Text:      text,
        ChunkSize: chunkSize,
        Strategy:  "word_boundary",
    }
    
    data, _ := json.Marshal(req)
    resp, err := http.Post("http://localhost:8080/chunk", "application/json", bytes.NewBuffer(data))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result ChunkResponse
    json.NewDecoder(resp.Body).Decode(&result)
    return &result, nil
}
```
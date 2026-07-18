# Error Handling

Comprehensive error reference for the Chunker API and CLI.

## Error Response Structure

All errors return a consistent JSON structure:

```json
{
  "error": "Human-readable error message",
  "code": "machine_readable_error_code"
}
```

| Field   | Type     | Description                              |
|---------|----------|------------------------------------------|
| `error` | `string` | Human-readable description of the error  |
| `code`  | `string` | Machine-readable error code for handling |

## HTTP Status Codes

| Code | Description                    | When It Occurs                           |
|------|--------------------------------|------------------------------------------|
| `200`| OK                            | Successful chunking operation            |
| `400`| Bad Request                   | Validation failed or invalid input       |
| `405`| Method Not Allowed            | Non-POST request to `/chunk` endpoint    |
| `500`| Internal Server Error         | Server-side failure (should not occur)   |

## Validation Errors

All validation errors return HTTP `400` status with `invalid_request` error code.

### Missing Required Fields

| Field        | Error Message                     | Error Code          | Recovery                                                                 |
|--------------|-----------------------------------|---------------------|--------------------------------------------------------------------------|
| `text`       | `text is required`                | `invalid_request`   | Provide the `text` field with your content to chunk                      |
| `chunk_size` | `chunk_size must be greater than 0`| `invalid_request` | Set `chunk_size` to a positive integer (minimum 1)                      |

#### Example

**Request:**
```json
{
  "chunk_size": 100
}
```

**Response (400):**
```json
{
  "error": "text is required",
  "code": "invalid_request"
}
```

---

### Invalid Value Constraints

| Constraint           | Error Message                           | Error Code          | Recovery                                                                      |
|----------------------|-----------------------------------------|---------------------|-------------------------------------------------------------------------------|
| Negative chunk size  | `chunk_size must be greater than 0`     | `invalid_request`   | Set `chunk_size` to a positive integer                                        |
| Negative overlap     | `overlap must be greater than or equal to 0`| `invalid_request` | Set `overlap` to 0 (no overlap) or a positive value for context overlap       |
| Overlap >= size      | `overlap must be less than chunk_size`  | `invalid_request`   | Reduce overlap to be strictly less than chunk_size, or increase chunk_size   |

#### Example

**Request:**
```json
{
  "text": "Sample text",
  "chunk_size": 100,
  "overlap": 150
}
```

**Response (400):**
```json
{
  "error": "overlap must be less than chunk_size",
  "code": "invalid_request"
}
```

---

### Invalid Strategy

| Strategy      | Error Message                              | Error Code          | Recovery                                                                  |
|---------------|--------------------------------------------|---------------------|---------------------------------------------------------------------------|
| Unknown value | `failed to create chunker: unknown strategy: <value>` | `invalid_request`   | Use a valid strategy: `smart_boundary`, `sentence_boundary`, `word_boundary`, `paragraph_aware`, `markdown_aware`, `hard_cut`, or `token_based` |

#### Example

**Request:**
```json
{
  "text": "Sample text",
  "chunk_size": 100,
  "strategy": "invalid_strategy"
}
```

**Response (400):**
```json
{
  "error": "failed to create chunker: unknown strategy: invalid_strategy",
  "code": "invalid_request"
}
```

---

## Request Format Errors

### Invalid JSON

| Error                       | Error Code          | Recovery                                                   |
|-----------------------------|---------------------|------------------------------------------------------------|
| `Invalid request body`      | `invalid_request`   | Ensure request body is valid JSON (check syntax, quotes)   |

#### Example

**Request:**
```json
{
  "text": "Unclosed string
}
```

**Response (400):**
```json
{
  "error": "Invalid request body",
  "code": "invalid_request"
}
```

---

### Wrong HTTP Method

| Method     | Error Message          | Error Code              | Recovery                     |
|------------|-----------------------|-------------------------|------------------------------|
| GET, PUT, etc | `Method not allowed` | `method_not_allowed`    | Use POST to `/chunk` endpoint |

#### Example

**Request:**
```bash
GET /chunk HTTP/1.1
```

**Response (405):**
```json
{
  "error": "Method not allowed",
  "code": "method_not_allowed"
}
```

---

## CLI Errors

The CLI reports errors to stderr with non-zero exit codes.

### Input Errors

| Error                          | Exit Code | Recovery                                                     |
|--------------------------------|-----------|--------------------------------------------------------------|
| `error reading input: <detail>` | 1         | Check stdin is available and not empty                       |
| `no input text provided`       | 1         | Pipe text to stdin (e.g., `cat file.txt | chunker`)         |

### Processing Errors

| Error                              | Exit Code | Recovery                                                             |
|------------------------------------|-----------|----------------------------------------------------------------------|
| `error chunking text: <detail>`    | 1         | Check validation rules above (chunk_size, overlap, strategy)        |

#### Example CLI Error

```bash
$ echo "" | chunker -size 100
error chunking text: no input text provided
```

---

## Error Handling Best Practices

### Client-Side Validation

Before sending requests, validate locally to avoid unnecessary API calls:

```python
def validate_chunk_request(text: str, chunk_size: int, overlap: int) -> tuple[bool, str]:
    if not text:
        return False, "text is required"
    if chunk_size <= 0:
        return False, "chunk_size must be greater than 0"
    if overlap < 0:
        return False, "overlap must be greater than or equal to 0"
    if overlap >= chunk_size:
        return False, "overlap must be less than chunk_size"
    return True, ""
```

### Retry Strategy

- **400 errors**: Do NOT retry - fix the request payload
- **405 errors**: Do NOT retry - fix the HTTP method
- **500 errors**: May retry with exponential backoff (should not occur in normal operation)

### Logging Recommendations

Log at minimum:
- Error code and message
- Request parameters (sanitized)
- Timestamp
- HTTP status code

```json
{
  "timestamp": "2025-01-18T23:47:00Z",
  "error_code": "invalid_request",
  "error_message": "overlap must be less than chunk_size",
  "request_params": {
    "chunk_size": 100,
    "overlap": 150,
    "strategy": "smart_boundary"
  },
  "status_code": 400
}
```

---

## Error Code Reference

| Error Code            | HTTP Status | Category         | Description                          |
|-----------------------|-------------|------------------|--------------------------------------|
| `invalid_request`     | 400         | Validation       | Request failed validation rules      |
| `method_not_allowed`  | 405         | HTTP Method      | Incorrect HTTP method used           |
| `internal_error`      | 500         | Server           | Unexpected server failure            |

package domain

import "github.com/dotcommander/reliquary/chunking"

// ToLibStrategy maps a domain Strategy to the library Strategy. String values
// are identical, so this is a direct cast; centralized so callers don't import
// the lib's Strategy type directly.
func ToLibStrategy(s Strategy) chunking.Strategy {
	return chunking.Strategy(s)
}

// FromLibChunk projects a library chunk into the wire DTO, intentionally
// dropping Path/Metadata/ContentHash so the JSON response shape (snake_case,
// documented in API.md) is unchanged by the migration.
func FromLibChunk(c chunking.Chunk) Chunk {
	return Chunk{
		ID:         c.ID,
		StartChar:  c.StartChar,
		EndChar:    c.EndChar,
		Text:       c.Text,
		CharCount:  c.CharCount,
		WordCount:  c.WordCount,
		TokenCount: c.TokenCount,
	}
}

// FromLibChunks maps a slice of library chunks to wire DTOs.
func FromLibChunks(in []chunking.Chunk) []Chunk {
	out := make([]Chunk, len(in))
	for i := range in {
		out[i] = FromLibChunk(in[i])
	}
	return out
}

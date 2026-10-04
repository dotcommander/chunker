package domain

import "github.com/dotcommander/reliquary/chunking"

// ToLibStrategy maps a domain Strategy to the library Strategy. String values
// are identical, so this is a direct cast; centralized so callers don't import
// the lib's Strategy type directly.
func ToLibStrategy(s Strategy) chunking.Strategy {
	return chunking.Strategy(s)
}

// FromLibChunk projects a library chunk into the wire DTO, preserving the
// strategy-produced annotations: Path (heading breadcrumb), Metadata
// (markdown block annotations), and ContentHash (16-hex SHA-256 of Text).
func FromLibChunk(c chunking.Chunk) Chunk {
	return Chunk{
		ID:          c.ID,
		StartChar:   c.StartChar,
		EndChar:     c.EndChar,
		Text:        c.Text,
		CharCount:   c.CharCount,
		WordCount:   c.WordCount,
		TokenCount:  c.TokenCount,
		Path:        c.Path,
		Metadata:    c.Metadata,
		ContentHash: c.ContentHash,
	}
}

// SemanticOptsFrom maps wire semantic options to the library type. Zero
// fields stay zero: the library applies its own defaults for unset values.
func SemanticOptsFrom(o *SemanticOptions) chunking.SemanticOpts {
	if o == nil {
		return chunking.SemanticOpts{}
	}
	return chunking.SemanticOpts{
		BreakSensitivity: o.BreakSensitivity,
		MinChunkChars:    o.MinChunkChars,
		MaxChunkChars:    o.MaxChunkChars,
		SmoothingWindow:  o.SmoothingWindow,
		CoherenceWindow:  o.CoherenceWindow,
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

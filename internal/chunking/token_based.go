package chunking

import (
	"context"
	"fmt"
	"strings"

	"chunker/internal/domain"
	"github.com/pkoukk/tiktoken-go"
)

type TokenBasedChunker struct {
	baseChunker
}

func NewTokenBasedChunker() *TokenBasedChunker {
	return &TokenBasedChunker{
		baseChunker: baseChunker{strategy: domain.TokenBased},
	}
}

func (t *TokenBasedChunker) Chunk(ctx context.Context, text string, size int, overlap int) []domain.Chunk {
	// Note: size here represents max tokens, not characters
	if size <= 0 || text == "" {
		return []domain.Chunk{}
	}

	// Default to cl100k_base encoding
	encoding := string(domain.EncodingCL100K)
	return t.ChunkWithEncoding(ctx, text, size, overlap, encoding)
}

func (t *TokenBasedChunker) ChunkWithEncoding(ctx context.Context, text string, maxTokens int, overlap int, encodingName string) []domain.Chunk {
	enc, err := tiktoken.GetEncoding(encodingName)
	if err != nil {
		// Fall back to character-based chunking
		return t.fallbackChunk(ctx, text, maxTokens*4, overlap*4) // Approximate 4 chars per token
	}
	// tiktoken-go doesn't require Free()

	// Tokenize the entire text
	tokens := enc.Encode(text, nil, nil)
	if len(tokens) == 0 {
		return []domain.Chunk{}
	}

	var chunks []domain.Chunk
	chunkID := 0

	for i := 0; i < len(tokens); {
		start := i
		
		// Apply overlap from previous chunk
		if chunkID > 0 && overlap > 0 {
			start = i - overlap
			if start < 0 {
				start = 0
			}
		}
		
		end := i + maxTokens
		if end > len(tokens) {
			end = len(tokens)
		}
		
		// Extract chunk tokens
		chunkTokens := tokens[start:end]
		
		// Decode back to text
		chunkText := enc.Decode(chunkTokens)
		
		// Try to clean up chunk boundaries
		if end < len(tokens) {
			chunkText = t.cleanChunkBoundary(chunkText, false)
		}
		if start > 0 {
			chunkText = t.cleanChunkBoundary(chunkText, true)
		}
		
		if chunkText != "" {
			chunk := createChunk(chunkID, chunkText)
			chunk.TokenCount = len(chunkTokens)
			chunks = append(chunks, chunk)
			chunkID++
		}
		
		i = end
	}

	return chunks
}


func (t *TokenBasedChunker) cleanChunkBoundary(text string, isStart bool) string {
	if isStart {
		// Clean up the start of a chunk
		// Remove partial words at the beginning
		idx := strings.IndexFunc(text, func(r rune) bool {
			return r == ' ' || r == '\n' || r == '\t'
		})
		if idx > 0 && idx < 10 { // Likely a partial word
			return text[idx+1:]
		}
	} else {
		// Clean up the end of a chunk
		// Try to end at a sentence or word boundary
		lastSpace := strings.LastIndexAny(text, " \n\t")
		lastPeriod := strings.LastIndexAny(text, ".!?")
		
		if lastPeriod > 0 && lastPeriod > len(text)-20 {
			return text[:lastPeriod+1]
		} else if lastSpace > 0 && lastSpace > len(text)-10 {
			return text[:lastSpace]
		}
	}
	
	return text
}

func (t *TokenBasedChunker) fallbackChunk(ctx context.Context, text string, size int, overlap int) []domain.Chunk {
	// Fall back to word boundary chunker with character limits
	wordChunker := NewWordBoundaryChunker()
	return wordChunker.Chunk(ctx, text, size, overlap)
}


// CountTokens counts tokens for a given text and encoding
func CountTokens(text string, encoding string) (int, error) {
	enc, err := tiktoken.GetEncoding(encoding)
	if err != nil {
		return 0, fmt.Errorf("failed to get encoding: %w", err)
	}
	
	tokens := enc.Encode(text, nil, nil)
	return len(tokens), nil
}
package chunking

import (
	"context"
	"strings"

	"chunker/internal/domain"
)

type ParagraphAwareChunker struct {
	baseChunker
}

func NewParagraphAwareChunker() *ParagraphAwareChunker {
	return &ParagraphAwareChunker{
		baseChunker: baseChunker{strategy: domain.ParagraphAware},
	}
}

func (p *ParagraphAwareChunker) Chunk(ctx context.Context, text string, size int, overlap int) []domain.Chunk {
	if size <= 0 || text == "" {
		return []domain.Chunk{}
	}

	paragraphs := splitIntoParagraphs(text)
	var chunks []domain.Chunk
	var currentChunk strings.Builder
	chunkID := 0

	for i := 0; i < len(paragraphs); {
		currentChunk.Reset()
		
		// Add paragraphs until we exceed size
		for i < len(paragraphs) && currentChunk.Len()+len(paragraphs[i])+2 <= size {
			if currentChunk.Len() > 0 {
				currentChunk.WriteString("\n\n")
			}
			currentChunk.WriteString(paragraphs[i])
			i++
		}
		
		// If nothing was added, split the paragraph
		if currentChunk.Len() == 0 && i < len(paragraphs) {
			// Fall back to word boundary for large paragraphs
			wordChunker := NewWordBoundaryChunker()
			subChunks := wordChunker.Chunk(ctx, paragraphs[i], size, overlap)
			for _, sc := range subChunks {
				sc.ID = chunkID
				chunks = append(chunks, sc)
				chunkID++
			}
			i++
			continue
		}
		
		chunkText := strings.TrimSpace(currentChunk.String())
		if chunkText != "" {
			chunks = append(chunks, createChunk(chunkID, chunkText))
			chunkID++
		}
	}

	return chunks
}

func splitIntoParagraphs(text string) []string {
	// Split on double newlines or more
	parts := strings.Split(text, "\n\n")
	var paragraphs []string
	
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			paragraphs = append(paragraphs, trimmed)
		}
	}
	
	// If no paragraphs found, treat the whole text as one paragraph
	if len(paragraphs) == 0 && strings.TrimSpace(text) != "" {
		paragraphs = []string{strings.TrimSpace(text)}
	}
	
	return paragraphs
}
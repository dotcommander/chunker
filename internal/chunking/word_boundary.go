package chunking

import (
	"context"
	"strings"

	"chunker/internal/domain"
)

type WordBoundaryChunker struct {
	baseChunker
}

func NewWordBoundaryChunker() *WordBoundaryChunker {
	return &WordBoundaryChunker{
		baseChunker: baseChunker{strategy: domain.WordBoundary},
	}
}

func (w *WordBoundaryChunker) Chunk(ctx context.Context, text string, size int, overlap int) []domain.Chunk {
	if size <= 0 || text == "" {
		return []domain.Chunk{}
	}

	words := splitIntoWords(text)
	var chunks []domain.Chunk
	var currentChunk strings.Builder
	var overlapBuffer []string
	chunkID := 0

	for i := 0; i < len(words); {
		currentChunk.Reset()
		
		// Add overlap from previous chunk
		if chunkID > 0 && overlap > 0 && len(overlapBuffer) > 0 {
			overlapText := strings.Join(overlapBuffer, "")
			if len(overlapText) <= overlap {
				currentChunk.WriteString(overlapText)
			}
		}
		
		// Clear overlap buffer for new chunk
		overlapBuffer = nil
		startPos := currentChunk.Len()
		
		// Add words until we exceed size
		for i < len(words) && currentChunk.Len()+len(words[i]) <= size {
			currentChunk.WriteString(words[i])
			i++
		}
		
		// If nothing was added (word too long), add it anyway
		if currentChunk.Len() == startPos && i < len(words) {
			currentChunk.WriteString(words[i])
			i++
		}
		
		chunkText := strings.TrimSpace(currentChunk.String())
		if chunkText != "" {
			chunks = append(chunks, createChunk(chunkID, chunkText))
			
			// Prepare overlap buffer for next chunk
			if overlap > 0 {
				chunkWords := strings.Fields(chunkText)
				overlapSize := 0
				for j := len(chunkWords) - 1; j >= 0 && overlapSize < overlap; j-- {
					word := chunkWords[j]
					if overlapSize+len(word)+1 <= overlap {
						overlapBuffer = append([]string{word, " "}, overlapBuffer...)
						overlapSize += len(word) + 1
					}
				}
			}
			
			chunkID++
		}
	}

	return chunks
}
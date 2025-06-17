package chunking

import (
	"context"

	"chunker/internal/domain"
)

type HardCutChunker struct {
	baseChunker
}

func NewHardCutChunker() *HardCutChunker {
	return &HardCutChunker{
		baseChunker: baseChunker{strategy: domain.HardCut},
	}
}

func (h *HardCutChunker) Chunk(ctx context.Context, text string, size int, overlap int) []domain.Chunk {
	if size <= 0 || text == "" {
		return []domain.Chunk{}
	}

	var chunks []domain.Chunk
	runes := []rune(text)
	chunkID := 0
	
	for i := 0; i < len(runes); {
		end := i + size
		if end > len(runes) {
			end = len(runes)
		}
		
		// Apply overlap from previous chunk
		start := i
		if chunkID > 0 && overlap > 0 {
			start = i - overlap
			if start < 0 {
				start = 0
			}
		}
		
		chunkText := string(runes[start:end])
		chunks = append(chunks, createChunk(chunkID, chunkText))
		
		i = end
		chunkID++
	}

	return chunks
}
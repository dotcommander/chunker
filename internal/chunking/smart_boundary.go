package chunking

import (
	"context"
	"strings"

	"chunker/internal/domain"
	"github.com/jdkato/prose/v2"
)

type SmartBoundaryChunker struct {
	baseChunker
}

func NewSmartBoundaryChunker() *SmartBoundaryChunker {
	return &SmartBoundaryChunker{
		baseChunker: baseChunker{strategy: domain.SmartBoundary},
	}
}

func (s *SmartBoundaryChunker) Chunk(ctx context.Context, text string, size int, overlap int) []domain.Chunk {
	if size <= 0 || text == "" {
		return []domain.Chunk{}
	}

	// Use prose for advanced sentence detection
	doc, err := prose.NewDocument(text)
	if err != nil {
		// Fall back to basic sentence splitting
		return s.fallbackChunk(ctx, text, size, overlap)
	}

	sentences := doc.Sentences()
	if len(sentences) == 0 {
		return s.fallbackChunk(ctx, text, size, overlap)
	}

	var chunks []domain.Chunk
	var currentChunk strings.Builder
	var overlapSentences []string
	chunkID := 0

	for i := 0; i < len(sentences); {
		currentChunk.Reset()
		
		// Add overlap from previous chunk
		if chunkID > 0 && overlap > 0 && len(overlapSentences) > 0 {
			overlapText := strings.Join(overlapSentences, " ")
			if len(overlapText) <= overlap {
				currentChunk.WriteString(overlapText)
				currentChunk.WriteString(" ")
			}
		}
		
		overlapSentences = nil
		startLen := currentChunk.Len()
		
		// Add sentences until we exceed size
		for i < len(sentences) && currentChunk.Len()+len(sentences[i].Text) <= size {
			if currentChunk.Len() > startLen {
				currentChunk.WriteString(" ")
			}
			currentChunk.WriteString(sentences[i].Text)
			
			// Keep last few sentences for overlap
			if overlap > 0 {
				overlapSentences = append(overlapSentences, sentences[i].Text)
				// Limit overlap buffer
				totalLen := 0
				start := 0
				for j := len(overlapSentences) - 1; j >= 0; j-- {
					totalLen += len(overlapSentences[j])
					if totalLen > overlap {
						start = j + 1
						break
					}
				}
				if start > 0 {
					overlapSentences = overlapSentences[start:]
				}
			}
			
			i++
		}
		
		// If nothing was added, add at least one sentence
		if currentChunk.Len() == startLen && i < len(sentences) {
			currentChunk.WriteString(sentences[i].Text)
			overlapSentences = []string{sentences[i].Text}
			i++
		}
		
		chunkText := strings.TrimSpace(currentChunk.String())
		if chunkText != "" {
			chunk := createChunk(chunkID, chunkText)
			chunks = append(chunks, chunk)
			chunkID++
		}
	}

	return chunks
}

func (s *SmartBoundaryChunker) fallbackChunk(ctx context.Context, text string, size int, overlap int) []domain.Chunk {
	// Fall back to sentence boundary chunker
	sentChunker := NewSentenceBoundaryChunker()
	return sentChunker.Chunk(ctx, text, size, overlap)
}


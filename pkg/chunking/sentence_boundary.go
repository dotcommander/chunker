package chunking

import (
	"context"
	"strings"

	"chunker/internal/domain"
)

type SentenceBoundaryChunker struct {
	baseChunker
}

func NewSentenceBoundaryChunker() *SentenceBoundaryChunker {
	return &SentenceBoundaryChunker{
		baseChunker: baseChunker{strategy: domain.SentenceBoundary},
	}
}

func (s *SentenceBoundaryChunker) Chunk(ctx context.Context, text string, size int, overlap int) []domain.Chunk {
	if size <= 0 || text == "" {
		return []domain.Chunk{}
	}

	sentences := splitIntoSentences(text)
	var chunks []domain.Chunk
	var currentChunk strings.Builder
	var overlapSentences []string
	chunkID := 0

	for i := 0; i < len(sentences); {
		currentChunk.Reset()
		
		// Add overlap from previous chunk
		if chunkID > 0 && overlap > 0 && len(overlapSentences) > 0 {
			for _, sent := range overlapSentences {
				if currentChunk.Len()+len(sent) <= overlap {
					currentChunk.WriteString(sent)
					currentChunk.WriteString(" ")
				}
			}
		}
		
		overlapSentences = nil
		startLen := currentChunk.Len()
		
		// Add sentences until we exceed size
		for i < len(sentences) && currentChunk.Len()+len(sentences[i]) <= size {
			currentChunk.WriteString(sentences[i])
			if !strings.HasSuffix(sentences[i], " ") {
				currentChunk.WriteString(" ")
			}
			overlapSentences = append(overlapSentences, sentences[i])
			i++
		}
		
		// If nothing was added, add one sentence anyway
		if currentChunk.Len() == startLen && i < len(sentences) {
			currentChunk.WriteString(sentences[i])
			overlapSentences = []string{sentences[i]}
			i++
		}
		
		chunkText := strings.TrimSpace(currentChunk.String())
		if chunkText != "" {
			chunks = append(chunks, createChunk(chunkID, chunkText))
			chunkID++
		}
	}

	return chunks
}

func splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder
	
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		current.WriteRune(runes[i])
		
		// Check for sentence endings
		if isSentenceEnd(runes[i]) {
			// Look ahead for quotes or parentheses
			if i+1 < len(runes) && (runes[i+1] == '"' || runes[i+1] == '\'' || runes[i+1] == ')') {
				i++
				current.WriteRune(runes[i])
			}
			
			// Check if next char is space or end
			if i+1 >= len(runes) || runes[i+1] == ' ' || runes[i+1] == '\n' {
				sentences = append(sentences, strings.TrimSpace(current.String()))
				current.Reset()
			}
		}
	}
	
	if current.Len() > 0 {
		sentences = append(sentences, strings.TrimSpace(current.String()))
	}
	
	return sentences
}

func isSentenceEnd(r rune) bool {
	return r == '.' || r == '!' || r == '?'
}
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

// smartSplitParagraphs uses prose to detect paragraph boundaries more intelligently
func smartSplitParagraphs(text string) []string {
	// First try standard paragraph splitting
	paragraphs := splitIntoParagraphs(text)
	
	// If we get just one large paragraph, try to split it more intelligently
	if len(paragraphs) == 1 && len(paragraphs[0]) > 500 {
		doc, err := prose.NewDocument(paragraphs[0])
		if err == nil {
			sentences := doc.Sentences()
			
			// Group sentences into logical paragraphs based on topic shifts
			var smartParagraphs []string
			var currentPara strings.Builder
			
			for i, sent := range sentences {
				currentPara.WriteString(sent.Text)
				
				// Look for paragraph boundaries:
				// - Empty line indicators
				// - Major topic shifts (would need more NLP)
				// - Long gaps between sentences
				if i < len(sentences)-1 {
					currentPara.WriteString(" ")
					
					// Simple heuristic: if next sentence starts with certain patterns
					nextText := sentences[i+1].Text
					if startsNewParagraph(nextText) && currentPara.Len() > 100 {
						smartParagraphs = append(smartParagraphs, strings.TrimSpace(currentPara.String()))
						currentPara.Reset()
					}
				}
			}
			
			if currentPara.Len() > 0 {
				smartParagraphs = append(smartParagraphs, strings.TrimSpace(currentPara.String()))
			}
			
			if len(smartParagraphs) > 1 {
				return smartParagraphs
			}
		}
	}
	
	return paragraphs
}

func startsNewParagraph(text string) bool {
	// Common patterns that indicate a new paragraph
	patterns := []string{
		"However,", "Moreover,", "Furthermore,", "In addition,",
		"On the other hand,", "In conclusion,", "First,", "Second,",
		"Finally,", "Therefore,", "Thus,", "Nevertheless,",
	}
	
	for _, pattern := range patterns {
		if strings.HasPrefix(text, pattern) {
			return true
		}
	}
	
	// Check for numbered lists or bullet points
	if len(text) > 2 {
		r := []rune(text)
		if (r[0] >= '1' && r[0] <= '9' && (r[1] == '.' || r[1] == ')')) ||
			r[0] == '•' || r[0] == '-' || r[0] == '*' {
			return true
		}
	}
	
	return false
}
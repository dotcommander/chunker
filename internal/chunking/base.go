package chunking

import (
	"strings"
	"unicode"

	"chunker/internal/domain"
)

type baseChunker struct {
	strategy domain.Strategy
}

func (b *baseChunker) Strategy() domain.Strategy {
	return b.strategy
}

func countWords(text string) int {
	return len(strings.Fields(text))
}

func splitIntoWords(text string) []string {
	var words []string
	var currentWord strings.Builder
	
	for _, r := range text {
		if unicode.IsSpace(r) {
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
			words = append(words, string(r))
		} else {
			currentWord.WriteRune(r)
		}
	}
	
	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}
	
	return words
}

func createChunk(id int, text string) domain.Chunk {
	return domain.Chunk{
		ID:        id,
		Text:      text,
		CharCount: len(text),
		WordCount: countWords(text),
	}
}
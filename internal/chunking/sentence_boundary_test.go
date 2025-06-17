package chunking

import (
	"context"
	"testing"
)

func TestSentenceBoundaryChunker(t *testing.T) {
	chunker := NewSentenceBoundaryChunker()
	ctx := context.Background()

	tests := []struct {
		name      string
		text      string
		size      int
		overlap   int
		wantCount int
	}{
		{
			name:      "multiple sentences",
			text:      "First sentence. Second sentence. Third sentence.",
			size:      30,
			overlap:   0,
			wantCount: 3, // Each sentence is ~15-16 chars, so they'll be separate chunks
		},
		{
			name:      "single long sentence",
			text:      "This is a very long sentence that exceeds our chunk size limit.",
			size:      20,
			overlap:   0,
			wantCount: 1,
		},
		{
			name:      "sentences with different endings",
			text:      "Question? Exclamation! Statement.",
			size:      50,
			overlap:   0,
			wantCount: 1,
		},
		{
			name:      "sentence with quotes",
			text:      `He said "Hello." Then he left.`,
			size:      20,
			overlap:   0,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunker.Chunk(ctx, tt.text, tt.size, tt.overlap)
			if len(chunks) != tt.wantCount {
				t.Errorf("got %d chunks, want %d", len(chunks), tt.wantCount)
			}
		})
	}
}

func TestSplitIntoSentences(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "basic sentences",
			text:     "First. Second. Third.",
			expected: []string{"First.", "Second.", "Third."},
		},
		{
			name:     "mixed punctuation",
			text:     "Question? Exclamation! Statement.",
			expected: []string{"Question?", "Exclamation!", "Statement."},
		},
		{
			name:     "sentence with quotes",
			text:     `He said "Hello." Then left.`,
			expected: []string{`He said "Hello."`, "Then left."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences := splitIntoSentences(tt.text)
			if len(sentences) != len(tt.expected) {
				t.Errorf("got %d sentences, want %d", len(sentences), len(tt.expected))
				return
			}
			for i, sent := range sentences {
				if sent != tt.expected[i] {
					t.Errorf("sentence %d: got %q, want %q", i, sent, tt.expected[i])
				}
			}
		})
	}
}
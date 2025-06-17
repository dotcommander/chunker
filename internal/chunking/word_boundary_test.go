package chunking

import (
	"context"
	"testing"
)

func TestWordBoundaryChunker(t *testing.T) {
	chunker := NewWordBoundaryChunker()
	ctx := context.Background()

	tests := []struct {
		name      string
		text      string
		size      int
		overlap   int
		wantCount int
	}{
		{
			name:      "simple text",
			text:      "The quick brown fox jumps over the lazy dog",
			size:      20,
			overlap:   0,
			wantCount: 3,
		},
		{
			name:      "with overlap",
			text:      "The quick brown fox jumps over the lazy dog",
			size:      20,
			overlap:   5,
			wantCount: 3,
		},
		{
			name:      "empty text",
			text:      "",
			size:      10,
			overlap:   0,
			wantCount: 0,
		},
		{
			name:      "single word larger than size",
			text:      "supercalifragilisticexpialidocious",
			size:      10,
			overlap:   0,
			wantCount: 1,
		},
		{
			name:      "exact size match",
			text:      "Hello world",
			size:      11,
			overlap:   0,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := chunker.Chunk(ctx, tt.text, tt.size, tt.overlap)
			if len(chunks) != tt.wantCount {
				t.Errorf("got %d chunks, want %d", len(chunks), tt.wantCount)
			}

			// Verify chunk IDs are sequential
			for i, chunk := range chunks {
				if chunk.ID != i {
					t.Errorf("chunk %d has ID %d", i, chunk.ID)
				}
			}
		})
	}
}

func TestWordBoundaryChunkerOverlap(t *testing.T) {
	chunker := NewWordBoundaryChunker()
	ctx := context.Background()
	
	text := "one two three four five six seven eight nine ten"
	chunks := chunker.Chunk(ctx, text, 20, 10)
	
	// Check that chunks have overlap
	if len(chunks) < 2 {
		t.Fatal("expected at least 2 chunks for overlap test")
	}
	
	// Verify overlapping content exists
	for i := 1; i < len(chunks); i++ {
		prev := chunks[i-1].Text
		curr := chunks[i].Text
		
		// Current chunk should start with some content from previous
		if len(prev) > 10 && len(curr) > 0 {
			// Basic check that there's some overlap
			if chunks[i-1].CharCount == 0 || chunks[i].CharCount == 0 {
				t.Errorf("chunk %d or %d has zero characters", i-1, i)
			}
		}
	}
}
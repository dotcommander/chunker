package service

import (
	"context"
	"strings"
	"testing"

	"github.com/dotcommander/chunker/internal/domain"
)

func TestChunkService_Recursive_EnglishParagraphCascade(t *testing.T) {
	t.Parallel()

	resp, err := NewChunkService().ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:      "First paragraph.\n\nSecond paragraph.",
		ChunkSize: 20,
		Strategy:  domain.Recursive,
	})
	if err != nil {
		t.Fatalf("ProcessChunkRequest() unexpected error: %v", err)
	}
	if resp.Metadata.StrategyUsed != domain.Recursive {
		t.Errorf("strategy_used = %q, want recursive", resp.Metadata.StrategyUsed)
	}
	if len(resp.Chunks) != 2 {
		t.Fatalf("got %d chunks, want 2 (paragraph split): %+v", len(resp.Chunks), resp.Chunks)
	}
	if resp.Chunks[0].Text != "First paragraph." {
		t.Errorf("chunk 0 text = %q, want %q", resp.Chunks[0].Text, "First paragraph.")
	}
}

func TestChunkService_Recursive_CJKThaiProfile(t *testing.T) {
	t.Parallel()

	// Pure CJK prose: the default profile has no CJK separators, so it falls
	// back to rune cuts; cjk_thai splits at the ideographic full stop.
	const text = "第一句。第二句。第三句。"

	resp, err := NewChunkService().ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:             text,
		ChunkSize:        6,
		Strategy:         domain.Recursive,
		SeparatorProfile: domain.ProfileCJKThai,
	})
	if err != nil {
		t.Fatalf("ProcessChunkRequest() unexpected error: %v", err)
	}

	if resp.Metadata.SeparatorProfile != domain.ProfileCJKThai {
		t.Errorf("metadata.separator_profile = %q, want cjk_thai", resp.Metadata.SeparatorProfile)
	}
	got := chunkTexts(resp.Chunks)
	want := []string{"第一句", "。第二句", "。第三句"}
	if len(got) != len(want) {
		t.Fatalf("chunks = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("chunk %d text = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestChunkService_Recursive_DefaultProfileRuneCutsCJK(t *testing.T) {
	t.Parallel()

	resp, err := NewChunkService().ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:      "第一句。第二句。第三句。",
		ChunkSize: 6,
		Strategy:  domain.Recursive,
	})
	if err != nil {
		t.Fatalf("ProcessChunkRequest() unexpected error: %v", err)
	}

	if resp.Metadata.SeparatorProfile != "" {
		t.Errorf("metadata.separator_profile = %q, want empty (no profile requested)", resp.Metadata.SeparatorProfile)
	}
	got := chunkTexts(resp.Chunks)
	want := []string{"第一句。第二", "句。第三句。"}
	if len(got) != len(want) {
		t.Fatalf("chunks = %v, want %v (rune hard-cut fallback)", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("chunk %d text = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestChunkService_Semantic_UnavailableWithoutEmbedder(t *testing.T) {
	t.Parallel()

	_, err := NewChunkService().ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:      "Some text.",
		ChunkSize: 100,
		Strategy:  domain.Semantic,
	})
	if !domain.IsRequestValidationError(err) {
		t.Fatalf("error = %v, want a request validation error", err)
	}
	if !strings.Contains(err.Error(), "embedding") {
		t.Errorf("error = %v, want it to explain the embedding requirement", err)
	}
}

func chunkTexts(chunks []domain.Chunk) []string {
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Text
	}
	return texts
}

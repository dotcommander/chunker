package service

import (
	"context"
	"errors"
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

// uniformBatchEmbedder returns identical vectors for every input, so
// consecutive-sentence similarity is uniform and no topic breaks fire.
type uniformBatchEmbedder struct {
	calls int
}

func (u *uniformBatchEmbedder) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	u.calls++
	vectors := make([][]float32, len(texts))
	for i := range texts {
		vectors[i] = []float32{1, 1}
	}
	return vectors, nil
}

// failingBatchEmbedder fails every call, exercising the library's documented
// fallback to smart-boundary chunking on embedding failure.
type failingBatchEmbedder struct{}

func (failingBatchEmbedder) EmbedBatch(context.Context, []string) ([][]float32, error) {
	return nil, errors.New("embedding provider unavailable")
}

func TestChunkService_Semantic_UniformEmbeddingsSingleTopic(t *testing.T) {
	t.Parallel()

	embedder := &uniformBatchEmbedder{}
	svc := NewChunkServiceWithEmbedder(embedder)

	resp, err := svc.ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:      "First sentence about one topic. Second sentence on the same theme. Third sentence continues it.",
		ChunkSize: 500,
		Strategy:  domain.Semantic,
	})
	if err != nil {
		t.Fatalf("ProcessChunkRequest() unexpected error: %v", err)
	}
	if embedder.calls == 0 {
		t.Fatal("embedder was never called; semantic path did not run")
	}
	if resp.Metadata.StrategyUsed != domain.Semantic {
		t.Errorf("strategy_used = %q, want semantic", resp.Metadata.StrategyUsed)
	}
	// Uniform similarity means no detected topic boundary: one merged chunk.
	if len(resp.Chunks) != 1 {
		t.Fatalf("got %d chunks, want 1 for uniform similarity: %+v", len(resp.Chunks), chunkTexts(resp.Chunks))
	}
	if resp.Chunks[0].ContentHash == "" {
		t.Error("semantic chunk must carry content_hash")
	}
}

func TestChunkService_Semantic_EmbedderFailureFallsBack(t *testing.T) {
	t.Parallel()

	svc := NewChunkServiceWithEmbedder(failingBatchEmbedder{})

	resp, err := svc.ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:      "First sentence here. Second sentence follows. A third one as well.",
		ChunkSize: 30,
		Strategy:  domain.Semantic,
	})
	if err != nil {
		t.Fatalf("ProcessChunkRequest() unexpected error: %v (graceful fallback required)", err)
	}
	if len(resp.Chunks) == 0 {
		t.Fatal("fallback produced no chunks")
	}
	if resp.Metadata.StrategyUsed != domain.Semantic {
		t.Errorf("strategy_used = %q, want semantic (requested strategy)", resp.Metadata.StrategyUsed)
	}
}

func TestChunkService_Semantic_OptionsValidation(t *testing.T) {
	t.Parallel()

	svc := NewChunkServiceWithEmbedder(&uniformBatchEmbedder{})

	_, err := svc.ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:      "Some text.",
		ChunkSize: 100,
		Strategy:  domain.SmartBoundary,
		Semantic:  &domain.SemanticOptions{BreakSensitivity: 1.5},
	})
	if !domain.IsRequestValidationError(err) || !strings.Contains(err.Error(), "semantic options require") {
		t.Fatalf("misuse error = %v, want semantic-options/strategy mismatch", err)
	}

	_, err = svc.ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:      "Some text.",
		ChunkSize: 100,
		Strategy:  domain.Semantic,
		Semantic:  &domain.SemanticOptions{MinChunkChars: 500, MaxChunkChars: 100},
	})
	if !domain.IsRequestValidationError(err) || !strings.Contains(err.Error(), "max_chunk_chars") {
		t.Fatalf("bounds error = %v, want min/max mismatch", err)
	}
}

func chunkTexts(chunks []domain.Chunk) []string {
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Text
	}
	return texts
}

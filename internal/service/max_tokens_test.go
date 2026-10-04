package service

import (
	"context"
	"strings"
	"testing"

	"github.com/dotcommander/chunker/internal/domain"
)

const maxTokensText = "The quick brown fox jumps over the lazy dog. " +
	"Pack my box with five dozen liquor jugs. " +
	"How vexingly quick daft zebras jump!"

func TestChunkService_MaxTokens_SplitsOversizedChunks(t *testing.T) {
	t.Parallel()

	resp, err := NewChunkService().ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:      maxTokensText,
		ChunkSize: 500, // chars: the whole text fits one chunk before enforcement
		Strategy:  domain.SmartBoundary,
		MaxTokens: 8,
	})
	if err != nil {
		t.Fatalf("ProcessChunkRequest() unexpected error: %v", err)
	}

	if resp.Metadata.MaxTokens != 8 {
		t.Errorf("metadata.max_tokens = %d, want 8 (applied ceiling must be reported)", resp.Metadata.MaxTokens)
	}
	if resp.Metadata.TokenEncoding != domain.EncodingCL100K {
		t.Errorf("metadata.token_encoding = %q, want the effective encoding %q",
			resp.Metadata.TokenEncoding, domain.EncodingCL100K)
	}
	if len(resp.Chunks) < 2 {
		t.Fatalf("got %d chunks, want >= 2: the text exceeds 8 tokens and must be split", len(resp.Chunks))
	}
	for _, chunk := range resp.Chunks {
		if chunk.TokenCount <= 0 {
			t.Errorf("chunk %d: token_count = %d, want > 0 (counts must be filled under max_tokens)", chunk.ID, chunk.TokenCount)
		}
		if chunk.TokenCount > 8 {
			t.Errorf("chunk %d: token_count = %d, exceeds the 8-token ceiling", chunk.ID, chunk.TokenCount)
		}
		if strings.TrimSpace(chunk.Text) == "" {
			t.Errorf("chunk %d: whitespace-only chunk must not be emitted", chunk.ID)
		}
		if chunk.ContentHash == "" {
			t.Errorf("chunk %d: content_hash must be present after enforcement splits", chunk.ID)
		}
	}
}

func TestChunkService_MaxTokens_NoOpWhenCeilingGenerous(t *testing.T) {
	t.Parallel()

	resp, err := NewChunkService().ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:      "Hello world.",
		ChunkSize: 100,
		Strategy:  domain.SmartBoundary,
		MaxTokens: 500,
	})
	if err != nil {
		t.Fatalf("ProcessChunkRequest() unexpected error: %v", err)
	}

	if len(resp.Chunks) != 1 {
		t.Fatalf("got %d chunks, want 1: a generous ceiling must not split", len(resp.Chunks))
	}
	if resp.Chunks[0].TokenCount <= 0 {
		t.Errorf("token_count = %d, want > 0 (counts filled under max_tokens)", resp.Chunks[0].TokenCount)
	}
	if resp.Metadata.MaxTokens != 500 {
		t.Errorf("metadata.max_tokens = %d, want 500", resp.Metadata.MaxTokens)
	}
}

func TestChunkService_MaxTokens_RejectsNegative(t *testing.T) {
	t.Parallel()

	_, err := NewChunkService().ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:      "Hello world.",
		ChunkSize: 100,
		MaxTokens: -1,
	})
	if !domain.IsRequestValidationError(err) {
		t.Fatalf("error = %v, want a request validation error", err)
	}
	if !strings.Contains(err.Error(), "max_tokens") {
		t.Errorf("error = %v, want it to name max_tokens", err)
	}
}

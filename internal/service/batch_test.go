package service

import (
	"context"
	"strings"
	"testing"

	"github.com/dotcommander/chunker/internal/domain"
)

func TestChunkService_Batch_HappyPath(t *testing.T) {
	t.Parallel()

	svc := NewChunkService()
	resp, err := svc.ProcessBatchChunkRequest(context.Background(), domain.BatchChunkRequest{
		Texts:     []string{"First document text.", "Second document text.", "Third document text."},
		ChunkSize: 8,
		Overlap:   0,
		Strategy:  domain.WordBoundary,
	})
	if err != nil {
		t.Fatalf("ProcessBatchChunkRequest() unexpected error: %v", err)
	}
	if len(resp.Results) != 3 {
		t.Fatalf("got %d results, want 3", len(resp.Results))
	}
	for i, res := range resp.Results {
		if res.Index != i {
			t.Errorf("results[%d].Index = %d, want %d (ordering)", i, res.Index, i)
		}
		if res.Error != "" {
			t.Errorf("results[%d].Error = %q, want none", i, res.Error)
			continue
		}
		if res.Response == nil {
			t.Fatalf("results[%d].Response = nil", i)
		}
		if len(res.Response.Chunks) == 0 {
			t.Errorf("results[%d] has no chunks", i)
		}
		if res.Response.Metadata.StrategyUsed != domain.WordBoundary {
			t.Errorf("results[%d].strategy_used = %q, want word_boundary", i, res.Response.Metadata.StrategyUsed)
		}
	}
}

func TestChunkService_Batch_MixedValidAndInvalid(t *testing.T) {
	t.Parallel()

	svc := NewChunkService()
	resp, err := svc.ProcessBatchChunkRequest(context.Background(), domain.BatchChunkRequest{
		Texts:     []string{"Valid text here.", "", "Also valid."},
		ChunkSize: 100,
	})
	if err != nil {
		t.Fatalf("ProcessBatchChunkRequest() unexpected error: %v (item failures must not fail the batch)", err)
	}
	if len(resp.Results) != 3 {
		t.Fatalf("got %d results, want 3", len(resp.Results))
	}
	if resp.Results[0].Response == nil || resp.Results[0].Error != "" {
		t.Errorf("results[0] = %+v, want success", resp.Results[0])
	}
	if resp.Results[1].Response != nil || !strings.Contains(resp.Results[1].Error, "text is required") {
		t.Errorf("results[1] = %+v, want item error for empty text", resp.Results[1])
	}
	if resp.Results[2].Response == nil || resp.Results[2].Error != "" {
		t.Errorf("results[2] = %+v, want success", resp.Results[2])
	}
}

func TestChunkService_Batch_InternalErrorIsGenericPerItem(t *testing.T) {
	t.Parallel()

	// An embedder failure is 500-class; the batch item must carry the generic
	// message, never internal error text.
	svc := NewChunkServiceWithEmbedder(failingBatchEmbedder{})
	resp, err := svc.ProcessBatchChunkRequest(context.Background(), domain.BatchChunkRequest{
		Texts:             []string{"Some text."},
		ChunkSize:         100,
		IncludeEmbeddings: true,
	})
	if err != nil {
		t.Fatalf("ProcessBatchChunkRequest() unexpected error: %v", err)
	}
	if resp.Results[0].Error != "internal error" {
		t.Errorf("item error = %q, want generic \"internal error\"", resp.Results[0].Error)
	}
}

func TestChunkService_Batch_InvalidParamsFailWholeBatch(t *testing.T) {
	t.Parallel()

	svc := NewChunkService()
	_, err := svc.ProcessBatchChunkRequest(context.Background(), domain.BatchChunkRequest{
		Texts:     []string{"Valid text.", "Another."},
		ChunkSize: 50,
		Overlap:   50,
	})
	if !domain.IsRequestValidationError(err) {
		t.Fatalf("error = %v, want a request validation error", err)
	}
}

func TestChunkService_Batch_EmptyTextsRejected(t *testing.T) {
	t.Parallel()

	svc := NewChunkService()
	_, err := svc.ProcessBatchChunkRequest(context.Background(), domain.BatchChunkRequest{
		ChunkSize: 100,
	})
	if !domain.IsRequestValidationError(err) || !strings.Contains(err.Error(), "texts") {
		t.Fatalf("error = %v, want missing-texts validation error", err)
	}
}

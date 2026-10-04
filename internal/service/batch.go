package service

import (
	"context"
	"errors"
	"log"

	"github.com/dotcommander/chunker/internal/domain"
)

// ProcessBatchChunkRequest chunks each text under the batch's shared
// parameters, sequentially and independently. Item-level failures (empty
// text, per-text validation) become per-item error results; batch-level
// failures (shared params, context cancellation) fail the whole call.
func (s *ChunkService) ProcessBatchChunkRequest(ctx context.Context, req domain.BatchChunkRequest) (*domain.BatchChunkResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	results := make([]domain.BatchChunkResult, len(req.Texts))
	for i, text := range req.Texts {
		resp, err := s.ProcessChunkRequest(ctx, req.ToChunkRequest(text))
		switch {
		case err == nil:
			results[i] = domain.BatchChunkResult{Index: i, Response: resp}
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
			return nil, err
		case domain.IsRequestValidationError(err):
			results[i] = domain.BatchChunkResult{Index: i, Error: err.Error()}
		default:
			// Match the single-request convention: log internally, keep the
			// item payload generic so internals never leak to clients.
			log.Printf("ERROR: batch item %d failed: %v", i, err)
			results[i] = domain.BatchChunkResult{Index: i, Error: "internal error"}
		}
	}
	return &domain.BatchChunkResponse{Results: results}, nil
}

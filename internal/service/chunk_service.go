package service

import (
	"context"
	"fmt"

	"chunker/internal/domain"
	"chunker/pkg/chunking"
)

type ChunkService struct {
	factory domain.ChunkerFactory
}

func NewChunkService(factory domain.ChunkerFactory) *ChunkService {
	return &ChunkService{factory: factory}
}

func (s *ChunkService) ProcessChunkRequest(ctx context.Context, req domain.ChunkRequest) (*domain.ChunkResponse, error) {
	if req.Text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}
	
	if req.ChunkSize <= 0 {
		return nil, fmt.Errorf("chunk_size must be positive")
	}
	
	if req.Overlap < 0 {
		return nil, fmt.Errorf("overlap cannot be negative")
	}
	
	if req.Overlap >= req.ChunkSize {
		return nil, fmt.Errorf("overlap must be less than chunk_size")
	}
	
	// Get strategy with default
	strategy := req.Strategy.WithDefault()
	
	// Create chunker
	chunker, err := s.factory.CreateChunker(strategy)
	if err != nil {
		return nil, fmt.Errorf("failed to create chunker: %w", err)
	}
	
	// Handle token-based chunking with specific encoding
	var chunks []domain.Chunk
	if strategy == domain.TokenBased && req.TokenEncoding != "" {
		tokenChunker, ok := chunker.(domain.TokenChunker)
		if ok {
			encoding := string(req.TokenEncoding.WithDefault())
			chunks = tokenChunker.ChunkWithEncoding(ctx, req.Text, req.ChunkSize, req.Overlap, encoding)
		} else {
			chunks = chunker.Chunk(ctx, req.Text, req.ChunkSize, req.Overlap)
		}
	} else {
		chunks = chunker.Chunk(ctx, req.Text, req.ChunkSize, req.Overlap)
	}
	
	// Calculate totals
	totalChars := len(req.Text)
	totalTokens := 0
	
	// Sum up token counts if available
	if strategy == domain.TokenBased {
		for _, chunk := range chunks {
			totalTokens += chunk.TokenCount
		}
		// If no token counts but encoding specified, count total
		if totalTokens == 0 && req.TokenEncoding != "" {
			encoding := string(req.TokenEncoding.WithDefault())
			totalTokens, _ = chunking.CountTokens(req.Text, encoding)
		}
	}
	
	metadata := domain.Metadata{
		TotalChunks:   len(chunks),
		TotalChars:    totalChars,
		TotalTokens:   totalTokens,
		StrategyUsed:  strategy,
		TokenEncoding: req.TokenEncoding,
	}
	
	return &domain.ChunkResponse{
		Chunks:   chunks,
		Metadata: metadata,
	}, nil
}
package service

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/dotcommander/reliquary/chunking"

	"github.com/dotcommander/chunker/internal/domain"
)

type ChunkService struct{}

func NewChunkService() *ChunkService {
	return &ChunkService{}
}

func (s *ChunkService) ProcessChunkRequest(ctx context.Context, req domain.ChunkRequest) (*domain.ChunkResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}

	strategy := req.Strategy.WithDefault()

	var libChunker chunking.Chunker
	var err error
	if strategy == domain.TokenBased && req.TokenEncoding != "" {
		libChunker, err = chunking.NewTokenChunker(string(req.TokenEncoding.WithDefault()))
	} else {
		libChunker, err = chunking.NewChunker(domain.ToLibStrategy(strategy))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create chunker: %w", err)
	}

	libChunks := libChunker.Chunk(req.Text, req.ChunkSize, req.Overlap)
	chunks := domain.FromLibChunks(libChunks)
	annotateChunkOffsets(req.Text, chunks)

	totalChars := utf8.RuneCountInString(req.Text)
	totalTokens, err := calculateTotalTokens(req, strategy)
	if err != nil {
		return nil, fmt.Errorf("count tokens: %w", err)
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

// annotateChunkOffsets converts the library's authoritative byte offsets
// (source[StartChar:EndChar] == Text) into rune/character offsets, matching
// the rune basis of TotalChars and the start_char/end_char wire contract.
// The library clears a span to (0,0) when it cannot map a chunk back to the
// source; invalid spans are normalized to (0,0) rather than re-derived by
// substring search, which previously discarded valid spans and emitted -1.
func annotateChunkOffsets(source string, chunks []domain.Chunk) {
	for i := range chunks {
		startByte, endByte := chunks[i].StartChar, chunks[i].EndChar
		if startByte < 0 || endByte < 0 || endByte < startByte || endByte > len(source) {
			chunks[i].StartChar = 0
			chunks[i].EndChar = 0
			continue
		}
		chunks[i].StartChar = utf8.RuneCountInString(source[:startByte])
		chunks[i].EndChar = utf8.RuneCountInString(source[:endByte])
	}
}

// calculateTotalTokens returns the token count of the ORIGINAL request text
// for token_based chunking, on the same basis as TotalChars (original text,
// not a sum across chunks). Summing per-chunk counts would double-count the
// overlap region; counting the source text once is consistent with TotalChars.
func calculateTotalTokens(req domain.ChunkRequest, strategy domain.Strategy) (int, error) {
	if strategy != domain.TokenBased {
		return 0, nil
	}
	encoding := string(req.TokenEncoding.WithDefault())
	return chunking.CountTokens(req.Text, encoding)
}

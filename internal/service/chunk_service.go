package service

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/dotcommander/reliquary/chunking"

	"github.com/dotcommander/chunker/internal/domain"
)

// ChunkService chunks text. The optional embedder enables the semantic
// strategy and include_embeddings payloads; without it, both fail
// validation clearly.
type ChunkService struct {
	embedder       chunking.BatchEmbedder
	embeddingModel string
}

func NewChunkService() *ChunkService {
	return &ChunkService{}
}

// NewChunkServiceWithEmbedder returns a service whose semantic strategy and
// include_embeddings payloads are backed by the given embedder.
func NewChunkServiceWithEmbedder(embedder chunking.BatchEmbedder) *ChunkService {
	return &ChunkService{embedder: embedder}
}

// WithEmbeddingModel records the provider model name reported in metadata
// when responses embed chunk text. Chainable for wiring convenience.
func (s *ChunkService) WithEmbeddingModel(model string) *ChunkService {
	s.embeddingModel = model
	return s
}

func (s *ChunkService) ProcessChunkRequest(ctx context.Context, req domain.ChunkRequest) (*domain.ChunkResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}
	if req.IncludeEmbeddings && s.embedder == nil {
		return nil, domain.EmbeddingsUnavailable()
	}

	strategy := req.Strategy.WithDefault()

	var libChunks []chunking.Chunk
	if strategy == domain.Semantic {
		// The semantic strategy needs an embedding provider. Without one, an
		// explicit request must fail clearly rather than fall back; with one,
		// embedding failures fall back to smart boundary inside the library.
		if s.embedder == nil {
			return nil, domain.SemanticUnavailable()
		}
		semanticChunker, err := chunking.NewSemanticChunker(s.embedder, domain.SemanticOptsFrom(req.Semantic))
		if err != nil {
			return nil, fmt.Errorf("failed to create semantic chunker: %w", err)
		}
		libChunks = semanticChunker.ChunkSemantic(ctx, req.Text, req.ChunkSize, req.Overlap)
	} else {
		libChunker, err := s.buildChunker(strategy, req)
		if err != nil {
			return nil, err
		}
		libChunks = libChunker.Chunk(req.Text, req.ChunkSize, req.Overlap)
	}
	effectiveEncoding := req.TokenEncoding
	var err error
	if req.MaxTokens > 0 {
		effectiveEncoding = req.TokenEncoding.WithDefault()
		libChunks, err = enforceTokenCeiling(libChunks, string(effectiveEncoding), req.MaxTokens)
		if err != nil {
			return nil, err
		}
	}
	chunks := domain.FromLibChunks(libChunks)
	annotateChunkOffsets(req.Text, chunks)

	totalChars := utf8.RuneCountInString(req.Text)
	totalTokens, err := calculateTotalTokens(req, strategy)
	if err != nil {
		return nil, fmt.Errorf("count tokens: %w", err)
	}

	metadata := domain.Metadata{
		TotalChunks:      len(chunks),
		TotalChars:       totalChars,
		TotalTokens:      totalTokens,
		StrategyUsed:     strategy,
		TokenEncoding:    effectiveEncoding,
		MaxTokens:        req.MaxTokens,
		SeparatorProfile: req.SeparatorProfile,
	}
	if req.IncludeEmbeddings {
		if err := s.embedChunks(ctx, chunks, &metadata); err != nil {
			return nil, err
		}
	}

	return &domain.ChunkResponse{
		Chunks:   chunks,
		Metadata: metadata,
	}, nil
}

// embedChunks embeds every chunk's final text in one batch call and attaches
// the vectors in order. Called after token-limit composition so the embedded
// text is exactly what the response carries. An explicit include_embeddings
// request fails on embedder error — no silent fallback.
func (s *ChunkService) embedChunks(ctx context.Context, chunks []domain.Chunk, metadata *domain.Metadata) error {
	if len(chunks) == 0 {
		return nil
	}
	texts := make([]string, len(chunks))
	for i, c := range chunks {
		texts[i] = c.Text
	}
	vectors, err := s.embedder.EmbedBatch(ctx, texts)
	if err != nil {
		return fmt.Errorf("embed chunks: %w", err)
	}
	if len(vectors) != len(chunks) {
		return fmt.Errorf("embed chunks: provider returned %d vectors for %d chunks", len(vectors), len(chunks))
	}
	for i, v := range vectors {
		chunks[i].Embedding = v
	}
	metadata.EmbeddingModel = s.embeddingModel
	metadata.EmbeddingDimensions = len(vectors[0])
	return nil
}

// buildChunker constructs the non-semantic strategy chunker.
func (s *ChunkService) buildChunker(strategy domain.Strategy, req domain.ChunkRequest) (chunking.Chunker, error) {
	switch {
	case strategy == domain.TokenBased && req.TokenEncoding != "":
		return chunking.NewTokenChunker(string(req.TokenEncoding.WithDefault()))
	case strategy == domain.Recursive && req.SeparatorProfile == domain.ProfileCJKThai:
		return chunking.NewRecursiveChunker(chunking.CJKThaiSeparatorProfile()), nil
	default:
		libChunker, err := chunking.NewChunker(domain.ToLibStrategy(strategy))
		if err != nil {
			return nil, fmt.Errorf("failed to create chunker: %w", err)
		}
		return libChunker, nil
	}
}

// enforceTokenCeiling splits any chunk whose token count exceeds maxTokens
// (sentence → word → hard-cut cascade in reliquary) and fills per-chunk
// TokenCount so responses report the effect of the applied ceiling.
func enforceTokenCeiling(chunks []chunking.Chunk, encoding string, maxTokens int) ([]chunking.Chunk, error) {
	tokenizer, err := chunking.NewTiktokenTokenizer(encoding)
	if err != nil {
		return nil, fmt.Errorf("failed to create tokenizer: %w", err)
	}
	limited, err := chunking.EnforceTokenLimitsWithTokenizer(chunks, tokenizer, maxTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to enforce max_tokens: %w", err)
	}
	if err := chunking.FillTokenCountsWithTokenizer(limited, tokenizer); err != nil {
		return nil, fmt.Errorf("failed to count tokens: %w", err)
	}
	return limited, nil
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

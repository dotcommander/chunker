package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/dotcommander/chunker/internal/domain"
	"github.com/dotcommander/reliquary/embedding"
)

func TestBuildBatchEmbedder_ConfigValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     EmbeddingConfig
		wantErr string
	}{
		{
			name:    "empty provider",
			cfg:     EmbeddingConfig{},
			wantErr: "embedding provider is not configured",
		},
		{
			name:    "unknown provider",
			cfg:     EmbeddingConfig{Provider: "vertex"},
			wantErr: `unknown embedding provider: "vertex"`,
		},
		{
			name:    "openai missing model",
			cfg:     EmbeddingConfig{Provider: "openai", Dimensions: 1536},
			wantErr: "embedding.model is required for provider openai",
		},
		{
			name:    "openai missing dimensions",
			cfg:     EmbeddingConfig{Provider: "openai", Model: "text-embedding-3-small"},
			wantErr: "embedding.dimensions must be greater than 0",
		},
		{
			name:    "openai missing api key",
			cfg:     EmbeddingConfig{Provider: "openai", Model: "text-embedding-3-small", Dimensions: 1536, APIKeyEnv: "CHUNKER_TEST_MISSING_KEY"},
			wantErr: "environment variable CHUNKER_TEST_MISSING_KEY is not set",
		},
		{
			name:    "ollama missing base url",
			cfg:     EmbeddingConfig{Provider: "ollama", Model: "nomic-embed-text"},
			wantErr: "embedding.base_url is required for provider ollama",
		},
		{
			name:    "ollama missing model",
			cfg:     EmbeddingConfig{Provider: "ollama", BaseURL: "http://localhost:11434"},
			wantErr: "embedding.model is required for provider ollama",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := BuildBatchEmbedder(tt.cfg)
			if err == nil {
				t.Fatalf("BuildBatchEmbedder() error = nil, want containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("BuildBatchEmbedder() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestBuildBatchEmbedder_OpenAIAcceptsValidConfig(t *testing.T) {
	// Not parallel: t.Setenv mutates process env.
	t.Setenv("CHUNKER_TEST_OPENAI_KEY", "test-key")

	embedder, err := BuildBatchEmbedder(EmbeddingConfig{
		Provider:   "openai",
		Model:      "text-embedding-3-small",
		Dimensions: 1536,
		APIKeyEnv:  "CHUNKER_TEST_OPENAI_KEY",
		MaxBatch:   8,
	})
	if err != nil {
		t.Fatalf("BuildBatchEmbedder() unexpected error: %v", err)
	}
	if embedder == nil {
		t.Fatal("BuildBatchEmbedder() returned nil embedder")
	}
}

// fakeEmbedder counts embedding requests and returns one unit vector per input.
type fakeEmbedder struct {
	requests []embedding.Request
}

func (f *fakeEmbedder) Embed(_ context.Context, req embedding.Request) (embedding.Result, error) {
	f.requests = append(f.requests, req)
	vectors := make([]embedding.Vector, len(req.Inputs))
	for i := range req.Inputs {
		vectors[i] = embedding.Vector{float32(i), 1}
	}
	return embedding.Result{Model: req.Model, Vectors: vectors}, nil
}

func TestBatchLimitedEmbedder_SplitsOversizedBatches(t *testing.T) {
	t.Parallel()

	inner := &fakeEmbedder{}
	limited := wrapBatchLimit(inner, 4)

	inputs := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	result, err := limited.Embed(t.Context(), embedding.Request{Inputs: inputs})
	if err != nil {
		t.Fatalf("Embed() unexpected error: %v", err)
	}
	if len(result.Vectors) != len(inputs) {
		t.Fatalf("got %d vectors, want %d", len(result.Vectors), len(inputs))
	}
	for i, v := range result.Vectors {
		if len(v) != 2 || v[0] != float32(i%4) {
			t.Errorf("vector %d = %v, want per-batch index %d", i, v, i%4)
		}
	}
	// 10 inputs with max 4 → batches of 4, 4, 2.
	if len(inner.requests) != 3 {
		t.Fatalf("got %d requests, want 3 (4+4+2)", len(inner.requests))
	}
	for i, req := range inner.requests {
		wantLen := 4
		if i == 2 {
			wantLen = 2
		}
		if len(req.Inputs) != wantLen {
			t.Errorf("request %d has %d inputs, want %d", i, len(req.Inputs), wantLen)
		}
	}
}

func TestBatchLimitedEmbedder_UnderLimitSingleRequest(t *testing.T) {
	t.Parallel()

	inner := &fakeEmbedder{}
	limited := wrapBatchLimit(inner, 64)

	if _, err := limited.Embed(t.Context(), embedding.Request{Inputs: []string{"a", "b"}}); err != nil {
		t.Fatalf("Embed() unexpected error: %v", err)
	}
	if len(inner.requests) != 1 {
		t.Fatalf("got %d requests, want 1", len(inner.requests))
	}
}

type errEmbedder struct{ err error }

func (e *errEmbedder) Embed(context.Context, embedding.Request) (embedding.Result, error) {
	return embedding.Result{}, e.err
}

func TestBatchLimitedEmbedder_ErrorPassthrough(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("provider down")
	limited := wrapBatchLimit(&errEmbedder{err: sentinel}, 1)

	_, err := limited.Embed(t.Context(), embedding.Request{Inputs: []string{"a", "b"}})
	if !errors.Is(err, sentinel) {
		t.Fatalf("Embed() error = %v, want %v", err, sentinel)
	}
}

// indexBatchEmbedder returns a 3-dim vector per input whose first component
// is the input index, letting tests pin vector-to-chunk ordering.
type indexBatchEmbedder struct{}

func (indexBatchEmbedder) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))
	for i := range texts {
		vectors[i] = []float32{float32(i), 1, 1}
	}
	return vectors, nil
}

type shortBatchEmbedder struct{}

func (shortBatchEmbedder) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	if len(texts) > 1 {
		return [][]float32{{1, 1, 1}}, nil // wrong count on purpose
	}
	return [][]float32{{1, 1, 1}}, nil
}

func TestChunkService_IncludeEmbeddings_VectorPerChunk(t *testing.T) {
	t.Parallel()

	svc := NewChunkServiceWithEmbedder(indexBatchEmbedder{}).WithEmbeddingModel("test-model")

	resp, err := svc.ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:              "First sentence here. Second sentence follows. A third one as well. A fourth closes it out.",
		ChunkSize:         30,
		Overlap:           0,
		Strategy:          domain.SmartBoundary,
		IncludeEmbeddings: true,
	})
	if err != nil {
		t.Fatalf("ProcessChunkRequest() unexpected error: %v", err)
	}
	if len(resp.Chunks) < 2 {
		t.Fatalf("got %d chunks, want at least 2 to exercise ordering", len(resp.Chunks))
	}
	for i, c := range resp.Chunks {
		if len(c.Embedding) != 3 {
			t.Fatalf("chunk %d embedding dims = %d, want 3", i, len(c.Embedding))
		}
		if c.Embedding[0] != float32(i) {
			t.Errorf("chunk %d embedding = %v, want first component %d (vector-to-chunk ordering)", i, c.Embedding, i)
		}
	}
	if resp.Metadata.EmbeddingDimensions != 3 {
		t.Errorf("metadata.embedding_dimensions = %d, want 3", resp.Metadata.EmbeddingDimensions)
	}
	if resp.Metadata.EmbeddingModel != "test-model" {
		t.Errorf("metadata.embedding_model = %q, want test-model", resp.Metadata.EmbeddingModel)
	}
}

func TestChunkService_IncludeEmbeddings_RequiresEmbedder(t *testing.T) {
	t.Parallel()

	_, err := NewChunkService().ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:              "Some text.",
		ChunkSize:         100,
		Strategy:          domain.SmartBoundary,
		IncludeEmbeddings: true,
	})
	if !domain.IsRequestValidationError(err) {
		t.Fatalf("error = %v, want a request validation error", err)
	}
	if !strings.Contains(err.Error(), "include_embeddings") {
		t.Errorf("error = %v, want it to name include_embeddings", err)
	}
}

func TestChunkService_IncludeEmbeddings_FailureSurfaced(t *testing.T) {
	t.Parallel()

	svc := NewChunkServiceWithEmbedder(failingBatchEmbedder{})

	resp, err := svc.ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:              "Some text to chunk.",
		ChunkSize:         100,
		Strategy:          domain.SmartBoundary,
		IncludeEmbeddings: true,
	})
	if err == nil {
		t.Fatalf("error = nil, want surfaced embedder failure (no silent fallback); resp = %+v", resp)
	}
	if !strings.Contains(err.Error(), "embed chunks") {
		t.Errorf("error = %v, want embed-chunks context", err)
	}
}

func TestChunkService_IncludeEmbeddings_VectorCountMismatch(t *testing.T) {
	t.Parallel()

	svc := NewChunkServiceWithEmbedder(shortBatchEmbedder{})

	_, err := svc.ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:              "First sentence here. Second sentence follows.",
		ChunkSize:         20,
		Overlap:           0,
		Strategy:          domain.SmartBoundary,
		IncludeEmbeddings: true,
	})
	if err == nil || !strings.Contains(err.Error(), "vectors for") {
		t.Fatalf("error = %v, want a vector/chunk count mismatch error", err)
	}
}

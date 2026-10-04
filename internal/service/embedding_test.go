package service

import (
	"context"
	"errors"
	"strings"
	"testing"

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

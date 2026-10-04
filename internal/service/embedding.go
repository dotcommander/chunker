package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/dotcommander/reliquary/adapter/ollama"
	"github.com/dotcommander/reliquary/adapter/openai"
	"github.com/dotcommander/reliquary/chunking"
	"github.com/dotcommander/reliquary/embedding"
	openaisdk "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// EmbeddingConfig configures the semantic-strategy embedder. All values are
// config data (~/.config/chunker/config.yaml, embedding: section); nothing
// provider-specific is hardcoded here beyond constructor wiring.
type EmbeddingConfig struct {
	// Provider enables semantic chunking when set: "openai" or "ollama".
	// Empty disables the embedder; the semantic strategy then returns a
	// clear validation error.
	Provider string
	BaseURL  string // required for ollama; optional OpenAI-compatible endpoint override
	Model    string // required for both providers
	// APIKeyEnv names the environment variable carrying the OpenAI API key
	// (default OPENAI_API_KEY). Ignored by ollama.
	APIKeyEnv string
	// Dimensions is required and must be > 0 for openai (the adapter always
	// sends the parameter); optional for ollama (0 = model default).
	Dimensions int
	// TimeoutMs bounds each provider request (default 30000).
	TimeoutMs int
	// MaxBatch splits embedding requests into batches of at most this many
	// inputs (0 = no limit).
	MaxBatch int
}

const defaultEmbeddingTimeout = 30 * time.Second

// BuildBatchEmbedder constructs the configured provider embedder and adapts
// it to the BatchEmbedder seam semantic chunking consumes.
func BuildBatchEmbedder(cfg EmbeddingConfig) (chunking.BatchEmbedder, error) {
	model := embedding.ModelRef{Provider: cfg.Provider, Name: cfg.Model, Dim: cfg.Dimensions}
	timeout := defaultEmbeddingTimeout
	if cfg.TimeoutMs > 0 {
		timeout = time.Duration(cfg.TimeoutMs) * time.Millisecond
	}

	var embedder embedding.Embedder
	var err error
	switch cfg.Provider {
	case "openai":
		embedder, err = buildOpenAIEmbedder(cfg, model, timeout)
	case "ollama":
		embedder, err = buildOllamaEmbedder(cfg, model, timeout)
	case "":
		return nil, fmt.Errorf("embedding provider is not configured")
	default:
		return nil, fmt.Errorf("unknown embedding provider: %q", cfg.Provider)
	}
	if err != nil {
		return nil, err
	}

	return chunking.BatchEmbedderFromEmbedder(wrapBatchLimit(embedder, cfg.MaxBatch), model, embedding.KindDocument), nil
}

func buildOpenAIEmbedder(cfg EmbeddingConfig, model embedding.ModelRef, timeout time.Duration) (embedding.Embedder, error) {
	if cfg.Model == "" {
		return nil, fmt.Errorf("embedding.model is required for provider openai")
	}
	if cfg.Dimensions <= 0 {
		return nil, fmt.Errorf("embedding.dimensions must be greater than 0 for provider openai (e.g. 1536 for text-embedding-3-small)")
	}
	apiKeyEnv := cfg.APIKeyEnv
	if apiKeyEnv == "" {
		apiKeyEnv = "OPENAI_API_KEY"
	}
	apiKey := os.Getenv(apiKeyEnv)
	if apiKey == "" {
		return nil, fmt.Errorf("environment variable %s is not set (embedding.api_key_env)", apiKeyEnv)
	}

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithHTTPClient(&http.Client{Timeout: timeout}),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	return openai.New(openaisdk.NewClient(opts...), openai.Config{Model: cfg.Model, Dimensions: cfg.Dimensions})
}

func buildOllamaEmbedder(cfg EmbeddingConfig, model embedding.ModelRef, timeout time.Duration) (embedding.Embedder, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("embedding.base_url is required for provider ollama (e.g. http://localhost:11434)")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("embedding.model is required for provider ollama")
	}
	return ollama.New(&http.Client{Timeout: timeout}, ollama.Config{
		BaseURL:    cfg.BaseURL,
		Model:      cfg.Model,
		Dimensions: cfg.Dimensions,
	})
}

// wrapBatchLimit returns an embedder that splits oversized input batches into
// requests of at most max inputs and reassembles the vectors in order.
func wrapBatchLimit(e embedding.Embedder, max int) embedding.Embedder {
	if max <= 0 {
		return e
	}
	return &batchLimitedEmbedder{inner: e, max: max}
}

type batchLimitedEmbedder struct {
	inner embedding.Embedder
	max   int
}

func (b *batchLimitedEmbedder) Embed(ctx context.Context, req embedding.Request) (embedding.Result, error) {
	if len(req.Inputs) <= b.max {
		return b.inner.Embed(ctx, req)
	}
	var out []embedding.Vector
	for start := 0; start < len(req.Inputs); start += b.max {
		end := min(start+b.max, len(req.Inputs))
		part := req
		part.Inputs = req.Inputs[start:end]
		res, err := b.inner.Embed(ctx, part)
		if err != nil {
			return embedding.Result{}, err
		}
		if len(res.Vectors) != end-start {
			return embedding.Result{}, embedding.ErrInvalidResult
		}
		out = append(out, res.Vectors...)
	}
	return embedding.Result{Model: req.Model, Vectors: out}, nil
}

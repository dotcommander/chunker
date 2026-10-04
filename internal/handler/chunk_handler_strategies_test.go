package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dotcommander/chunker/internal/service"
)

// Strategy round-trips through the full HTTP path with the real service,
// pinning status mapping for the recursive/semantic strategies and
// separator_profile validation.

func TestChunkHandler_HandleChunk_RecursiveRoundTrip(t *testing.T) {
	t.Parallel()

	handler := NewChunkHandler(service.NewChunkService())

	reqBody := `{"text": "First paragraph.\n\nSecond paragraph.", "chunk_size": 20, "strategy": "recursive", "separator_profile": "cjk_thai"}`
	req := httptest.NewRequest(http.MethodPost, "/chunk", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.HandleChunk(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %v, want %v; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if body := w.Body.String(); !strings.Contains(body, `"strategy_used":"recursive"`) {
		t.Errorf("body = %q, want strategy_used recursive", body)
	}
	if body := w.Body.String(); !strings.Contains(body, `"separator_profile":"cjk_thai"`) {
		t.Errorf("body = %q, want metadata to echo the separator profile", body)
	}
}

func TestChunkHandler_HandleChunk_UnknownSeparatorProfile(t *testing.T) {
	t.Parallel()

	handler := NewChunkHandler(service.NewChunkService())

	reqBody := `{"text": "Hello", "chunk_size": 100, "strategy": "recursive", "separator_profile": "bogus"}`
	req := httptest.NewRequest(http.MethodPost, "/chunk", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.HandleChunk(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %v, want %v", w.Code, http.StatusBadRequest)
	}
	if body := w.Body.String(); !strings.Contains(body, "separator_profile") {
		t.Errorf("body = %q, want it to name separator_profile", body)
	}
}

func TestChunkHandler_HandleChunk_SemanticWithoutEmbedder(t *testing.T) {
	t.Parallel()

	handler := NewChunkHandler(service.NewChunkService())

	reqBody := `{"text": "Hello world.", "chunk_size": 100, "strategy": "semantic"}`
	req := httptest.NewRequest(http.MethodPost, "/chunk", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.HandleChunk(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %v, want %v", w.Code, http.StatusBadRequest)
	}
	if body := w.Body.String(); !strings.Contains(body, "embedding") {
		t.Errorf("body = %q, want the embedding requirement explained", body)
	}
}

// uniformBatchEmbedder mirrors the service fake; kept local to the handler
// package so this test wires only public constructors.
type handlerUniformEmbedder struct{}

func (handlerUniformEmbedder) EmbedBatch(_ context.Context, texts []string) ([][]float32, error) {
	vectors := make([][]float32, len(texts))
	for i := range texts {
		vectors[i] = []float32{1, 1}
	}
	return vectors, nil
}

func TestChunkHandler_HandleChunk_SemanticWithEmbedder(t *testing.T) {
	t.Parallel()

	handler := NewChunkHandler(service.NewChunkServiceWithEmbedder(handlerUniformEmbedder{}))

	reqBody := `{"text": "One topic sentence. Same topic continues.", "chunk_size": 500, "strategy": "semantic", "semantic": {"break_sensitivity": 1.25}}`
	req := httptest.NewRequest(http.MethodPost, "/chunk", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.HandleChunk(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %v, want %v; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if body := w.Body.String(); !strings.Contains(body, `"strategy_used":"semantic"`) {
		t.Errorf("body = %q, want strategy_used semantic", body)
	}
}

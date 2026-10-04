package handler

import (
	"bytes"
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

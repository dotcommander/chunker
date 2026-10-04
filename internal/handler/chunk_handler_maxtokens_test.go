package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dotcommander/chunker/internal/service"
)

// Validation of max_tokens happens in the service; this test pins the full
// HTTP path (decode → service validation → 400) with the real service wired,
// so the status mapping cannot drift from the validation boundary.

func TestChunkHandler_HandleChunk_MaxTokensNegative(t *testing.T) {
	t.Parallel()

	handler := NewChunkHandler(service.NewChunkService())

	reqBody := `{"text": "Hello world", "chunk_size": 100, "max_tokens": -1}`
	req := httptest.NewRequest(http.MethodPost, "/chunk", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.HandleChunk(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleChunk() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
	if body := w.Body.String(); !strings.Contains(body, "max_tokens") {
		t.Errorf("HandleChunk() body = %q, want it to name max_tokens", body)
	}
}

func TestChunkHandler_HandleChunk_MaxTokensAccepted(t *testing.T) {
	t.Parallel()

	handler := NewChunkHandler(service.NewChunkService())

	reqBody := `{"text": "Hello world.", "chunk_size": 100, "max_tokens": 500}`
	req := httptest.NewRequest(http.MethodPost, "/chunk", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.HandleChunk(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("HandleChunk() status = %v, want %v; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	if body := w.Body.String(); !strings.Contains(body, `"max_tokens":500`) {
		t.Errorf("HandleChunk() body = %q, want metadata to report the applied ceiling", body)
	}
}

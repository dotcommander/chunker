package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dotcommander/chunker/internal/domain"
)

// mockChunkService implements domain.ChunkService for testing
type mockChunkService struct {
	processFunc func(ctx context.Context, req domain.ChunkRequest) (*domain.ChunkResponse, error)
}

func (m *mockChunkService) ProcessChunkRequest(ctx context.Context, req domain.ChunkRequest) (*domain.ChunkResponse, error) {
	if m.processFunc != nil {
		return m.processFunc(ctx, req)
	}
	return &domain.ChunkResponse{
		Chunks: []domain.Chunk{
			{ID: 1, Text: "test", CharCount: 4, WordCount: 1},
		},
		Metadata: domain.Metadata{
			TotalChunks:  1,
			TotalChars:   4,
			StrategyUsed: domain.SmartBoundary,
		},
	}, nil
}

func TestChunkHandler_HandleChunk_Success(t *testing.T) {
	t.Parallel()
	mockSvc := &mockChunkService{}
	handler := NewChunkHandler(mockSvc)

	reqBody := `{"text": "Hello world", "chunk_size": 100}`
	req := httptest.NewRequest(http.MethodPost, "/chunk", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.HandleChunk(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleChunk() status = %v, want %v", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("HandleChunk() content-type = %v, want application/json", contentType)
	}

	var resp domain.ChunkResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Chunks) != 1 {
		t.Errorf("HandleChunk() chunks count = %v, want 1", len(resp.Chunks))
	}
}

func TestChunkHandler_HandleChunk_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	mockSvc := &mockChunkService{}
	handler := NewChunkHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/chunk", nil)
	w := httptest.NewRecorder()

	handler.HandleChunk(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("HandleChunk() status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestChunkHandler_HandleChunk_InvalidJSON(t *testing.T) {
	t.Parallel()
	mockSvc := &mockChunkService{}
	handler := NewChunkHandler(mockSvc)

	reqBody := `{"text": "unclosed`
	req := httptest.NewRequest(http.MethodPost, "/chunk", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.HandleChunk(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleChunk() status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error != "Invalid request body" {
		t.Errorf("HandleChunk() error = %v, want 'Invalid request body'", errResp.Error)
	}
}

func TestChunkHandler_HandleChunk_ServiceError(t *testing.T) {
	t.Parallel()
	mockSvc := &mockChunkService{
		processFunc: func(ctx context.Context, req domain.ChunkRequest) (*domain.ChunkResponse, error) {
			return nil, errors.New("chunk_size must be greater than 0")
		},
	}
	handler := NewChunkHandler(mockSvc)

	reqBody := `{"text": "test", "chunk_size": 0}`
	req := httptest.NewRequest(http.MethodPost, "/chunk", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.HandleChunk(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("HandleChunk() status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error != "chunk_size must be greater than 0" {
		t.Errorf("HandleChunk() error = %v, want validation error", errResp.Error)
	}
}

// TestChunkHandler_HandleChunk_BodyTooLarge verifies that a request body
// exceeding maxChunkRequestBytes is rejected with 413 before reaching the
// service, preventing unbounded memory growth in the JSON decoder.
func TestChunkHandler_HandleChunk_BodyTooLarge(t *testing.T) {
	t.Parallel()
	mockSvc := &mockChunkService{
		processFunc: func(_ context.Context, _ domain.ChunkRequest) (*domain.ChunkResponse, error) {
			t.Fatal("service should not be invoked for oversized body")
			return nil, nil
		},
	}
	handler := NewChunkHandler(mockSvc)

	// Build a body larger than maxChunkRequestBytes: a valid JSON envelope
	// containing a single oversized "text" field.
	oversized := bytes.Repeat([]byte("a"), maxChunkRequestBytes+1024)
	reqBody := bytes.NewBuffer(nil)
	reqBody.WriteString(`{"text":"`)
	reqBody.Write(oversized)
	reqBody.WriteString(`","chunk_size":100}`)

	req := httptest.NewRequest(http.MethodPost, "/chunk", reqBody)
	w := httptest.NewRecorder()

	handler.HandleChunk(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("HandleChunk() oversized status = %v, want %v", w.Code, http.StatusRequestEntityTooLarge)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if errResp.Error != "Request body too large" {
		t.Errorf("HandleChunk() oversized error = %q, want %q", errResp.Error, "Request body too large")
	}
}

func TestChunkHandler_HandleHealth(t *testing.T) {
	t.Parallel()
	mockSvc := &mockChunkService{}
	handler := NewChunkHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HandleHealth() status = %v, want %v", w.Code, http.StatusOK)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("HandleHealth() status = %v, want 'ok'", resp.Status)
	}
}

func TestSendJSON_EncodingError(t *testing.T) {
	t.Parallel()
	// Encoding errors are logged; can't assert without mocking logger.
	w := httptest.NewRecorder()

	// Create an invalid data structure that can't be encoded
	invalidData := make(chan int)

	// This should log an error but not panic
	sendJSON(w, invalidData, http.StatusOK)

	// Verify status code was still written
	if w.Code != http.StatusOK {
		t.Errorf("sendJSON() status = %v, want %v", w.Code, http.StatusOK)
	}
}

func TestSendError(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	sendError(w, "test error", http.StatusBadRequest)

	if w.Code != http.StatusBadRequest {
		t.Errorf("sendError() status = %v, want %v", w.Code, http.StatusBadRequest)
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error != "test error" {
		t.Errorf("sendError() error = %v, want 'test error'", errResp.Error)
	}
}

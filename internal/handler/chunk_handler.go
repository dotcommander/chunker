package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/dotcommander/chunker/internal/domain"
)

// maxChunkRequestBytes caps the JSON request body for POST /chunk. Picked at
// 10 MiB to comfortably hold long-form documents users would reasonably chunk
// while preventing a single request from exhausting server memory. Oversized
// requests surface as 413 Payload Too Large rather than silently OOMing.
const maxChunkRequestBytes = 10 * 1024 * 1024

// ErrorResponse represents a typed error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// HealthResponse represents a typed health check response
type HealthResponse struct {
	Status string `json:"status"`
}

type ChunkHandler struct {
	service domain.ChunkService
}

func NewChunkHandler(service domain.ChunkService) *ChunkHandler {
	return &ChunkHandler{service: service}
}

func (h *ChunkHandler) HandleChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Cap request body before decoding. MaxBytesReader sets the response to
	// 413 via *http.MaxBytesError once the limit is exceeded; we distinguish
	// that from malformed JSON so clients get an actionable status.
	r.Body = http.MaxBytesReader(w, r.Body, maxChunkRequestBytes)

	var req domain.ChunkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			sendError(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.ProcessChunkRequest(r.Context(), req)
	if err != nil {
		sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	sendJSON(w, resp, http.StatusOK)
}

func (h *ChunkHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, HealthResponse{Status: "ok"}, http.StatusOK)
}

func sendJSON(w http.ResponseWriter, data any, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Status already written; log so monitoring can alert.
		log.Printf("ERROR: Failed to encode JSON response: %v", err)
	}
}

func sendError(w http.ResponseWriter, message string, status int) {
	sendJSON(w, ErrorResponse{Error: message}, status)
}

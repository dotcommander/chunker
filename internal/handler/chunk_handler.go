package handler

import (
	"encoding/json"
	"net/http"

	"chunker/internal/domain"
)

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

	var req domain.ChunkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
	sendJSON(w, map[string]string{"status": "ok"}, http.StatusOK)
}

func sendJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func sendError(w http.ResponseWriter, message string, status int) {
	sendJSON(w, map[string]string{"error": message}, status)
}
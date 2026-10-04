package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/dotcommander/chunker/internal/service"
)

func TestChunkHandler_HandleChunkBatch_MixedValidAndInvalid(t *testing.T) {
	t.Parallel()

	handler := NewChunkHandler(service.NewChunkService())

	reqBody := `{
		"texts": ["First valid document.", "", "Third valid document."],
		"chunk_size": 100
	}`
	req := httptest.NewRequest(http.MethodPost, "/chunk/batch", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	handler.HandleChunkBatch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %v, want %v; body: %s", w.Code, http.StatusOK, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{
		`"index":0`, `"index":1`, `"index":2`,
		`"text is required"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q: %s", want, body)
		}
	}
	if strings.Count(body, `"response"`) != 2 {
		t.Errorf("body = %s, want exactly two successful item responses", body)
	}
}

func TestChunkHandler_HandleChunkBatch_MissingTexts(t *testing.T) {
	t.Parallel()

	handler := NewChunkHandler(service.NewChunkService())

	req := httptest.NewRequest(http.MethodPost, "/chunk/batch", bytes.NewBufferString(`{"chunk_size": 100}`))
	w := httptest.NewRecorder()

	handler.HandleChunkBatch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %v, want %v", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "texts") {
		t.Errorf("body = %q, want the texts requirement explained", w.Body.String())
	}
}

func TestChunkHandler_HandleChunkBatch_InvalidJSON(t *testing.T) {
	t.Parallel()

	handler := NewChunkHandler(service.NewChunkService())

	req := httptest.NewRequest(http.MethodPost, "/chunk/batch", bytes.NewBufferString(`{not json`))
	w := httptest.NewRecorder()

	handler.HandleChunkBatch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

func TestChunkHandler_HandleChunkBatch_MethodNotAllowed(t *testing.T) {
	t.Parallel()

	handler := NewChunkHandler(service.NewChunkService())

	req := httptest.NewRequest(http.MethodGet, "/chunk/batch", nil)
	w := httptest.NewRecorder()

	handler.HandleChunkBatch(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %v, want %v", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestChunkHandler_HandleChunkBatch_ConcurrentRequests(t *testing.T) {
	t.Parallel()

	handler := NewChunkHandler(service.NewChunkService())
	const workers = 16

	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			marker := fmt.Sprintf("marker-%d", w)
			reqBody := fmt.Sprintf(`{"texts": ["%s alpha text.", "%s beta text."], "chunk_size": 100}`, marker, marker)
			req := httptest.NewRequest(http.MethodPost, "/chunk/batch", bytes.NewBufferString(reqBody))
			rec := httptest.NewRecorder()

			handler.HandleChunkBatch(rec, req)

			if rec.Code != http.StatusOK {
				errs <- fmt.Errorf("worker %d: status = %d, body: %s", w, rec.Code, rec.Body.String())
				return
			}
			body := rec.Body.String()
			if strings.Count(body, `"response"`) != 2 || strings.Contains(body, `"error"`) {
				errs <- fmt.Errorf("worker %d: unexpected envelope: %s", w, body)
				return
			}
			// Each worker must see only its own markers — no cross-request
			// mixing under concurrency.
			for _, other := range []string{"alpha", "beta"} {
				if !strings.Contains(body, marker+" "+other) {
					errs <- fmt.Errorf("worker %d: body missing %q: %s", w, marker+" "+other, body)
					return
				}
			}
			for _, wrong := range []string{"marker-0 ", "marker-1 ", "marker-2 "} {
				if w < 3 && strings.Contains(body, wrong) && wrong != marker+" " {
					errs <- fmt.Errorf("worker %d: body contains foreign marker %q", w, wrong)
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

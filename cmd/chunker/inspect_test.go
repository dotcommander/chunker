package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/dotcommander/chunker/internal/domain"
)

// errWriter is an io.Writer that returns the configured error on every Write.
// Used to exercise the encode-failure propagation path in
// printInspectSummaryNDJSON without depending on os-level writer failures.
type errWriter struct{ err error }

func (w errWriter) Write(p []byte) (int, error) { return 0, w.err }

// TestPrintInspectSummaryNDJSON_EmptyChunks confirms the zero-chunks branch
// emits a single "inspect_summary" record with total_chunks=0 and returns
// no error.
func TestPrintInspectSummaryNDJSON_EmptyChunks(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	req := domain.ChunkRequest{Text: "x", ChunkSize: 100}
	resp := &domain.ChunkResponse{Chunks: nil}
	if err := printInspectSummaryNDJSON(&buf, req, resp, 0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"total_chunks":0`) {
		t.Errorf("output should report zero chunks: %q", out)
	}
	if !strings.Contains(out, `"type":"inspect_summary"`) {
		t.Errorf("output should be inspect_summary record: %q", out)
	}
}

// TestPrintInspectSummaryNDJSON_HappyPath confirms a non-empty response
// produces a summary record plus per-sample records and returns no error.
func TestPrintInspectSummaryNDJSON_HappyPath(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	req := domain.ChunkRequest{Text: "hello world", ChunkSize: 10, Overlap: 2}
	resp := &domain.ChunkResponse{
		Chunks: []domain.Chunk{
			{ID: 0, StartChar: 0, EndChar: 5, Text: "hello", CharCount: 5},
			{ID: 1, StartChar: 6, EndChar: 11, Text: "world", CharCount: 5},
		},
		Metadata: domain.Metadata{StrategyUsed: domain.WordBoundary},
	}
	if err := printInspectSummaryNDJSON(&buf, req, resp, 2); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"total_chunks":2`) {
		t.Errorf("output should report two chunks: %q", out)
	}
	if !strings.Contains(out, `"type":"inspect_sample"`) {
		t.Errorf("output should contain per-sample records: %q", out)
	}
}

// TestPrintInspectSummaryNDJSON_EncodeError_Empty is the regression gate:
// previously the empty-chunks branch silently discarded encode errors via
// `_ = enc.Encode(...)`. With the propagation fix, a writer that always
// errors MUST surface that error to the caller (so the CLI can exit non-zero).
func TestPrintInspectSummaryNDJSON_EncodeError_Empty(t *testing.T) {
	t.Parallel()
	want := errors.New("write failed")
	w := errWriter{err: want}
	req := domain.ChunkRequest{Text: "x", ChunkSize: 100}
	resp := &domain.ChunkResponse{Chunks: nil}
	err := printInspectSummaryNDJSON(w, req, resp, 0)
	if err == nil {
		t.Fatal("expected encode error, got nil")
	}
	if !strings.Contains(err.Error(), "write failed") {
		t.Errorf("error should wrap underlying writer error: %v", err)
	}
}

// TestPrintInspectSummaryNDJSON_EncodeError_Summary covers the summary
// record encode-failure path (non-empty chunks, first Encode call fails).
func TestPrintInspectSummaryNDJSON_EncodeError_Summary(t *testing.T) {
	t.Parallel()
	want := errors.New("summary write failed")
	w := errWriter{err: want}
	req := domain.ChunkRequest{Text: "hello", ChunkSize: 10}
	resp := &domain.ChunkResponse{
		Chunks: []domain.Chunk{
			{ID: 0, StartChar: 0, EndChar: 5, Text: "hello", CharCount: 5},
		},
		Metadata: domain.Metadata{StrategyUsed: domain.WordBoundary},
	}
	err := printInspectSummaryNDJSON(w, req, resp, 1)
	if err == nil {
		t.Fatal("expected encode error, got nil")
	}
	if !strings.Contains(err.Error(), "summary write failed") {
		t.Errorf("error should wrap underlying writer error: %v", err)
	}
}

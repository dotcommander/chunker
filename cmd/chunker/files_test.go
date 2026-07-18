package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dotcommander/chunker/internal/domain"
)

// stubChunkService satisfies domain.ChunkService with deterministic behaviour
// for batch-summary tests. failOn paths short-circuit with errStub from
// ProcessChunkRequest; everything else returns a minimal valid response.
type stubChunkService struct {
	failOn map[string]bool
}

var errStub = errors.New("stub failure")

func (s *stubChunkService) ProcessChunkRequest(_ context.Context, req domain.ChunkRequest) (*domain.ChunkResponse, error) {
	if s.failOn[req.Text] {
		return nil, errStub
	}
	return &domain.ChunkResponse{
		Chunks: []domain.Chunk{
			{ID: 0, Text: req.Text, TokenCount: len(req.Text)},
		},
		Metadata: domain.Metadata{TotalChunks: 1},
	}, nil
}

func writeTempFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func TestProcessBatchFiles_AllSuccess(t *testing.T) {
	dir := t.TempDir()
	outDir := t.TempDir()
	a := writeTempFile(t, dir, "a.txt", "alpha")
	b := writeTempFile(t, dir, "b.txt", "beta")

	svc := &stubChunkService{}
	var stderr bytes.Buffer
	processed, failures := processBatchFiles(svc, []string{a, b}, outDir, &stderr)
	if processed != 2 {
		t.Errorf("processed=%d want 2", processed)
	}
	if len(failures) != 0 {
		t.Errorf("failures=%v want none", failures)
	}
}

func TestProcessBatchFiles_PartialFail(t *testing.T) {
	dir := t.TempDir()
	outDir := t.TempDir()
	a := writeTempFile(t, dir, "a.txt", "alpha")
	b := writeTempFile(t, dir, "b.txt", "bad")
	c := writeTempFile(t, dir, "c.txt", "gamma")

	svc := &stubChunkService{failOn: map[string]bool{"bad": true}}
	var stderr bytes.Buffer
	processed, failures := processBatchFiles(svc, []string{a, b, c}, outDir, &stderr)
	if processed != 2 {
		t.Errorf("processed=%d want 2", processed)
	}
	if len(failures) != 1 || failures[0].path != b || !errors.Is(failures[0].err, errStub) {
		t.Errorf("failures=%v want one entry for %s with errStub", failures, b)
	}
	if !strings.Contains(stderr.String(), "skip "+b) {
		t.Errorf("stderr=%q expected skip line for %s", stderr.String(), b)
	}
}

func TestReadBoundedFile_RejectsOversize(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "big.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 200)), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	// Over the cap → ErrStdinTooLarge (not an unbounded read / OOM).
	if _, err := readBoundedFile(path, 100); !errors.Is(err, ErrStdinTooLarge) {
		t.Errorf("oversize err = %v, want ErrStdinTooLarge", err)
	}

	// Under the cap → full content returned.
	data, err := readBoundedFile(path, 300)
	if err != nil {
		t.Fatalf("undersize read err = %v, want nil", err)
	}
	if len(data) != 200 {
		t.Errorf("undersize read len = %d, want 200", len(data))
	}
}

func TestProcessBatchFiles_AllFail(t *testing.T) {
	dir := t.TempDir()
	outDir := t.TempDir()
	a := writeTempFile(t, dir, "a.txt", "bad")
	b := writeTempFile(t, dir, "b.txt", "bad")

	svc := &stubChunkService{failOn: map[string]bool{"bad": true}}
	var stderr bytes.Buffer
	processed, failures := processBatchFiles(svc, []string{a, b}, outDir, &stderr)
	if processed != 0 {
		t.Errorf("processed=%d want 0", processed)
	}
	if len(failures) != 2 {
		t.Errorf("failures=%d want 2 entries", len(failures))
	}
}

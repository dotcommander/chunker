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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestProcessBatchFiles_ReportsOutputPathCollision(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outDir := t.TempDir()
	firstDir := filepath.Join(dir, "first")
	secondDir := filepath.Join(dir, "second")
	if err := os.MkdirAll(firstDir, 0o755); err != nil {
		t.Fatalf("mkdir first: %v", err)
	}
	if err := os.MkdirAll(secondDir, 0o755); err != nil {
		t.Fatalf("mkdir second: %v", err)
	}
	first := writeTempFile(t, firstDir, "report.txt", "alpha")
	second := writeTempFile(t, secondDir, "report.txt", "beta")

	var stderr bytes.Buffer
	processed, failures := processBatchFiles(&stubChunkService{}, []string{first, second}, outDir, &stderr)
	if processed != 1 || len(failures) != 1 {
		t.Fatalf("processed=%d failures=%v, want 1 and one failure", processed, failures)
	}
	if failures[0].path != second || !strings.Contains(failures[0].err.Error(), "output path collision") {
		t.Fatalf("failure=%+v, want collision for %s", failures[0], second)
	}
	if !strings.Contains(stderr.String(), "skip "+second) {
		t.Fatalf("stderr=%q, want skip for second source", stderr.String())
	}
	output, err := os.ReadFile(filepath.Join(outDir, "report.json"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if !bytes.Contains(output, []byte("alpha")) || bytes.Contains(output, []byte("beta")) {
		t.Fatalf("output=%s, want first source preserved", output)
	}
}

func TestProcessBatchFiles_ReportsFilesystemAliasCollision(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outDir := t.TempDir()
	first := writeTempFile(t, dir, "first.txt", "alpha")
	second := writeTempFile(t, dir, "second.txt", "beta")
	firstOutput := filepath.Join(outDir, "first.json")
	secondOutput := filepath.Join(outDir, "second.json")
	if err := os.WriteFile(firstOutput, []byte("old"), 0o644); err != nil {
		t.Fatalf("seed first output: %v", err)
	}
	if err := os.Link(firstOutput, secondOutput); err != nil {
		t.Fatalf("link output aliases: %v", err)
	}

	var stderr bytes.Buffer
	processed, failures := processBatchFiles(&stubChunkService{}, []string{first, second}, outDir, &stderr)
	if processed != 1 || len(failures) != 1 {
		t.Fatalf("processed=%d failures=%v, want 1 and one failure", processed, failures)
	}
	if failures[0].path != second || !strings.Contains(failures[0].err.Error(), "output path collision") {
		t.Fatalf("failure=%+v, want filesystem collision for %s", failures[0], second)
	}
	for _, outputPath := range []string{firstOutput, secondOutput} {
		output, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("read %s: %v", outputPath, err)
		}
		if !bytes.Contains(output, []byte("alpha")) || bytes.Contains(output, []byte("beta")) {
			t.Fatalf("output %s=%s, want first source preserved", outputPath, output)
		}
	}
}

type closeErrorWriter struct {
	bytes.Buffer
	err error
}

func (w *closeErrorWriter) Close() error {
	return w.err
}

func TestWriteOutputAndClose_ReportsCloseError(t *testing.T) {
	t.Parallel()
	closeErr := errors.New("disk flush failed")
	output := &closeErrorWriter{err: closeErr}
	resp := &domain.ChunkResponse{Chunks: []domain.Chunk{{ID: 1, Text: "test"}}}

	err := writeOutputAndClose(&stubChunkService{}, resp, output)

	if !errors.Is(err, closeErr) {
		t.Fatalf("writeOutputAndClose() error = %v, want close error", err)
	}
}

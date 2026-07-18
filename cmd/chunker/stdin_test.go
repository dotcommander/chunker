package main

import (
	"bytes"
	"errors"
	"testing"
)

func TestReadAllBounded_ExactlyAtLimit(t *testing.T) {
	t.Parallel()
	const limit int64 = 32
	data := bytes.Repeat([]byte{'a'}, int(limit))
	got, err := readAllBounded(bytes.NewReader(data), limit)
	if err != nil {
		t.Fatalf("unexpected error at exactly-limit boundary: %v", err)
	}
	if int64(len(got)) != limit {
		t.Errorf("got %d bytes, want %d", len(got), limit)
	}
}

func TestReadAllBounded_OneByteOverLimit(t *testing.T) {
	t.Parallel()
	const limit int64 = 32
	data := bytes.Repeat([]byte{'a'}, int(limit)+1) // limit+1 → must error
	_, err := readAllBounded(bytes.NewReader(data), limit)
	if err == nil {
		t.Fatal("expected ErrStdinTooLarge at limit+1, got nil")
	}
	if !errors.Is(err, ErrStdinTooLarge) {
		t.Errorf("error chain missing ErrStdinTooLarge: %v", err)
	}
}

func TestReadAllBounded_BelowLimit(t *testing.T) {
	t.Parallel()
	const limit int64 = 1024
	data := []byte("hello world")
	got, err := readAllBounded(bytes.NewReader(data), limit)
	if err != nil {
		t.Fatalf("unexpected error below limit: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("got %q, want %q", got, data)
	}
}

// stdinLimit env tests share process state; t.Setenv is incompatible with
// t.Parallel, so these run sequentially.
func TestStdinLimit_DefaultsToMax(t *testing.T) {
	t.Setenv("CHUNKER_MAX_STDIN_BYTES", "")
	if got := stdinLimit(); got != MaxStdinBytes {
		t.Errorf("stdinLimit() = %d, want default %d", got, MaxStdinBytes)
	}
}

func TestStdinLimit_HonoursEnvOverride(t *testing.T) {
	t.Setenv("CHUNKER_MAX_STDIN_BYTES", "4096")
	if got := stdinLimit(); got != 4096 {
		t.Errorf("stdinLimit() = %d, want 4096", got)
	}
}

func TestStdinLimit_IgnoresInvalidEnv(t *testing.T) {
	t.Setenv("CHUNKER_MAX_STDIN_BYTES", "not-a-number")
	if got := stdinLimit(); got != MaxStdinBytes {
		t.Errorf("stdinLimit() = %d, want default fallback %d", got, MaxStdinBytes)
	}
}

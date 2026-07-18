package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Config tests share process state via env/working dir, so they cannot use
// t.Parallel — t.Setenv is incompatible with parallel subtests.

func TestLoadConfig_EmbeddedDefaultsParse(t *testing.T) {
	cfg := mustParseEmbeddedDefault()
	if cfg.ChunkSize <= 0 {
		t.Errorf("embedded ChunkSize must be positive, got %d", cfg.ChunkSize)
	}
	if cfg.Overlap < 0 {
		t.Errorf("embedded Overlap must be non-negative, got %d", cfg.Overlap)
	}
	if cfg.ServerPort == "" {
		t.Error("embedded ServerPort must be non-empty")
	}
}

func TestLoadConfig_FirstRunWritesDefault(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "chunker", "config.yaml")
	t.Setenv("CHUNKER_CONFIG", target)

	cfg := LoadConfig()

	// File must now exist with embedded default contents.
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("expected first-run config to be written at %s: %v", target, err)
	}
	if len(data) == 0 {
		t.Fatal("first-run config file is empty")
	}

	// Returned cfg must match the embedded default.
	want := mustParseEmbeddedDefault()
	if cfg != want {
		t.Errorf("first-run LoadConfig = %+v, want embedded default %+v", cfg, want)
	}
}

func TestLoadConfig_UserOverridesAreApplied(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "chunker", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte("chunk_size: 1234\noverlap: 56\nserver_port: \"9999\"\n")
	if err := os.WriteFile(target, body, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CHUNKER_CONFIG", target)

	cfg := LoadConfig()
	if cfg.ChunkSize != 1234 {
		t.Errorf("ChunkSize = %d, want 1234", cfg.ChunkSize)
	}
	if cfg.Overlap != 56 {
		t.Errorf("Overlap = %d, want 56", cfg.Overlap)
	}
	if cfg.ServerPort != "9999" {
		t.Errorf("ServerPort = %q, want 9999", cfg.ServerPort)
	}
}

func TestLoadConfig_PartialUserFileKeepsDefaults(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "chunker", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	// Only chunk_size set; overlap and server_port should fall back to embedded.
	if err := os.WriteFile(target, []byte("chunk_size: 2222\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CHUNKER_CONFIG", target)

	cfg := LoadConfig()
	want := mustParseEmbeddedDefault()
	if cfg.ChunkSize != 2222 {
		t.Errorf("ChunkSize = %d, want 2222 (user override)", cfg.ChunkSize)
	}
	if cfg.Overlap != want.Overlap {
		t.Errorf("Overlap = %d, want default %d", cfg.Overlap, want.Overlap)
	}
	if cfg.ServerPort != want.ServerPort {
		t.Errorf("ServerPort = %q, want default %q", cfg.ServerPort, want.ServerPort)
	}
}

func TestLoadConfig_OverlapZeroExplicit(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "chunker", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	// overlap: 0 explicitly set — must override embedded default.
	if err := os.WriteFile(target, []byte("overlap: 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CHUNKER_CONFIG", target)

	cfg := LoadConfig()
	if cfg.Overlap != 0 {
		t.Errorf("Overlap = %d, want explicit 0 from user file", cfg.Overlap)
	}
}

func TestLoadConfig_CorruptYAMLFallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "chunker", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte(":::not valid yaml:::\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CHUNKER_CONFIG", target)

	cfg := LoadConfig()
	want := mustParseEmbeddedDefault()
	if cfg != want {
		t.Errorf("corrupt YAML did not fall back to embedded default: got %+v, want %+v", cfg, want)
	}
}

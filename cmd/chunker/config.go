package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds chunker defaults loaded from ~/.config/chunker/config.yaml.
// Flags continue to override these values at runtime (flags > config > embedded).
type Config struct {
	ChunkSize  int    `yaml:"chunk_size"`
	Overlap    int    `yaml:"overlap"`
	ServerBind string `yaml:"server_bind"`
	ServerPort string `yaml:"server_port"`
}

//go:embed default_config.yaml
var embeddedDefaultConfig []byte

// configPath returns the resolved path to the user config file. Honours
// CHUNKER_CONFIG (full path) and falls back to $XDG_CONFIG_HOME or ~/.config.
func configPath() (string, error) {
	if p := os.Getenv("CHUNKER_CONFIG"); p != "" {
		return p, nil
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "chunker", "config.yaml"), nil
}

// LoadConfig returns the active Config. If the user file does not exist it is
// created from the embedded default and that default is returned. Read or
// parse failures fall back to the embedded default with no fatal error so a
// corrupt config never blocks the binary.
func LoadConfig() Config {
	cfg := mustParseEmbeddedDefault()
	path, err := configPath()
	if err != nil {
		return cfg
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// First run: persist the embedded default so users can customize.
			if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr == nil {
				_ = os.WriteFile(path, embeddedDefaultConfig, 0o644)
			}
		}
		return cfg
	}

	var loaded Config
	if yaml.Unmarshal(data, &loaded) != nil {
		return cfg
	}
	// Zero values fall back to the embedded default so a partial file
	// (e.g. only chunk_size set) still gets sensible defaults.
	if loaded.ChunkSize > 0 {
		cfg.ChunkSize = loaded.ChunkSize
	}
	if loaded.Overlap >= 0 && data != nil {
		// Overlap=0 is a legitimate user choice; only override default if the
		// key was present. Detect by re-parsing into a map.
		var rawKeys map[string]any
		if yaml.Unmarshal(data, &rawKeys) == nil {
			if _, ok := rawKeys["overlap"]; ok {
				cfg.Overlap = loaded.Overlap
			}
		}
	}
	if loaded.ServerPort != "" {
		cfg.ServerPort = loaded.ServerPort
	}
	if loaded.ServerBind != "" {
		cfg.ServerBind = loaded.ServerBind
	}
	return cfg
}

func mustParseEmbeddedDefault() Config {
	var c Config
	if err := yaml.Unmarshal(embeddedDefaultConfig, &c); err != nil {
		// Programming error: the embedded YAML must always parse.
		panic(fmt.Sprintf("chunker: embedded default config is invalid: %v", err))
	}
	return c
}

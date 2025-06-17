package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"chunker/internal/domain"
)

// CLIFlags represents the command-line arguments
type CLIFlags struct {
	ChunkSize     int
	Overlap       int
	Strategy      string
	TokenEncoding string
	OutputFormat  string
	Pretty        bool
}

// CLIRunner encapsulates the CLI application logic
type CLIRunner struct {
	chunkService domain.ChunkService
	in           io.Reader
	out          io.Writer
	errOut       io.Writer
}

// NewCLIRunner creates a new CLI runner with dependency injection
func NewCLIRunner(cs domain.ChunkService, in io.Reader, out, errOut io.Writer) *CLIRunner {
	return &CLIRunner{
		chunkService: cs,
		in:           in,
		out:          out,
		errOut:       errOut,
	}
}

// Run executes the CLI logic
func (r *CLIRunner) Run(ctx context.Context, flags CLIFlags) error {
	// Read all input
	inputBytes, err := io.ReadAll(r.in)
	if err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	inputText := string(inputBytes)
	if inputText == "" {
		return fmt.Errorf("no input text provided")
	}

	// Create request
	req := domain.ChunkRequest{
		Text:          inputText,
		ChunkSize:     flags.ChunkSize,
		Overlap:       flags.Overlap,
		Strategy:      domain.Strategy(flags.Strategy),
		TokenEncoding: domain.TokenEncoding(flags.TokenEncoding),
	}

	// Process request
	resp, err := r.chunkService.ProcessChunkRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("error chunking text: %w", err)
	}

	// Output results
	return r.writeOutput(resp, flags.OutputFormat, flags.Pretty)
}

// writeOutput handles different output formats
func (r *CLIRunner) writeOutput(resp *domain.ChunkResponse, format string, pretty bool) error {
	switch format {
	case "jsonl":
		return r.outputJSONL(resp)
	default:
		return r.outputJSON(resp, pretty)
	}
}

// outputJSON writes the response as JSON
func (r *CLIRunner) outputJSON(resp *domain.ChunkResponse, pretty bool) error {
	encoder := json.NewEncoder(r.out)
	if pretty {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(resp)
}

// outputJSONL writes chunks as JSON Lines format
func (r *CLIRunner) outputJSONL(resp *domain.ChunkResponse) error {
	encoder := json.NewEncoder(r.out)
	
	// First line: metadata
	if err := encoder.Encode(map[string]interface{}{
		"type":     "metadata",
		"metadata": resp.Metadata,
	}); err != nil {
		return err
	}

	// Subsequent lines: chunks
	for _, chunk := range resp.Chunks {
		if err := encoder.Encode(map[string]interface{}{
			"type":  "chunk",
			"chunk": chunk,
		}); err != nil {
			return err
		}
	}

	return nil
}
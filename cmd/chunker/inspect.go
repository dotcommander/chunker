package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/dotcommander/chunker/internal/domain"
)

func runInspectCommand(args []string) {
	inspectFlags := newChunkFlagSet("inspect")
	sample := inspectFlags.Int("sample", 2, "Number of chunk previews to display")
	inspectFormat := inspectFlags.String("inspect-format", "ndjson", "Inspect output format: ndjson, human")
	if err := inspectFlags.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	raw, err := readAllBounded(os.Stdin, stdinLimit())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read stdin: %v\n", err)
		os.Exit(1)
	}

	text := string(raw)
	if text == "" {
		fmt.Fprintln(os.Stderr, "inspect requires stdin text")
		os.Exit(1)
	}

	req := chunkRequestFromFlags(text)
	resp, err := newChunkService().ProcessChunkRequest(context.Background(), req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "inspect failed: %v\n", err)
		os.Exit(1)
	}

	if *inspectFormat == "human" {
		printInspectSummaryHuman(req, resp, *sample)
		return
	}

	if err := printInspectSummaryNDJSON(os.Stdout, req, resp, *sample); err != nil {
		fmt.Fprintf(os.Stderr, "inspect encode failed: %v\n", err)
		os.Exit(1)
	}
}

func printInspectSummaryHuman(req domain.ChunkRequest, resp *domain.ChunkResponse, sample int) {
	total := len(resp.Chunks)
	if total == 0 {
		fmt.Fprintln(os.Stdout, "No chunks produced")
		return
	}

	minChars := resp.Chunks[0].CharCount
	maxChars := resp.Chunks[0].CharCount
	sumChars := 0
	for _, c := range resp.Chunks {
		sumChars += c.CharCount
		if c.CharCount < minChars {
			minChars = c.CharCount
		}
		if c.CharCount > maxChars {
			maxChars = c.CharCount
		}
	}

	avg := float64(sumChars) / float64(total)
	fmt.Fprintf(os.Stdout, "Strategy: %s\n", resp.Metadata.StrategyUsed)
	fmt.Fprintf(os.Stdout, "Chunk size: %d  Overlap: %d\n", req.ChunkSize, req.Overlap)
	fmt.Fprintf(os.Stdout, "Total chunks: %d  Avg chars: %.1f  Min/Max: %d/%d\n", total, avg, minChars, maxChars)
	if resp.Metadata.TotalTokens > 0 {
		fmt.Fprintf(os.Stdout, "Total tokens: %d  Encoding: %s\n", resp.Metadata.TotalTokens, resp.Metadata.TokenEncoding)
	}

	if sample < 0 {
		sample = 0
	}
	if sample > total {
		sample = total
	}
	for i := 0; i < sample; i++ {
		preview := resp.Chunks[i].Text
		if len([]rune(preview)) > 120 {
			preview = truncateRunes(preview, 120) + "..."
		}
		fmt.Fprintf(os.Stdout, "[%d] (%d-%d) %s\n", i, resp.Chunks[i].StartChar, resp.Chunks[i].EndChar, preview)
	}
}

// printInspectSummaryNDJSON encodes the inspect summary and (optionally)
// per-chunk samples to w as NDJSON. Returns the first encoder error so the
// CLI can exit non-zero — silently dropping encode failures previously
// masked truncated/broken stdout pipes from machine consumers.
func printInspectSummaryNDJSON(w io.Writer, req domain.ChunkRequest, resp *domain.ChunkResponse, sample int) error {
	enc := json.NewEncoder(w)

	total := len(resp.Chunks)
	if total == 0 {
		return enc.Encode(map[string]any{"type": "inspect_summary", "total_chunks": 0})
	}

	minChars := resp.Chunks[0].CharCount
	maxChars := resp.Chunks[0].CharCount
	sumChars := 0
	for _, c := range resp.Chunks {
		sumChars += c.CharCount
		if c.CharCount < minChars {
			minChars = c.CharCount
		}
		if c.CharCount > maxChars {
			maxChars = c.CharCount
		}
	}

	avg := float64(sumChars) / float64(total)
	if err := enc.Encode(map[string]any{
		"type":           "inspect_summary",
		"strategy":       resp.Metadata.StrategyUsed,
		"chunk_size":     req.ChunkSize,
		"overlap":        req.Overlap,
		"total_chunks":   total,
		"avg_chars":      avg,
		"min_chars":      minChars,
		"max_chars":      maxChars,
		"total_tokens":   resp.Metadata.TotalTokens,
		"token_encoding": resp.Metadata.TokenEncoding,
	}); err != nil {
		return err
	}

	if sample < 0 {
		sample = 0
	}
	if sample > total {
		sample = total
	}

	for i := 0; i < sample; i++ {
		preview := resp.Chunks[i].Text
		if len([]rune(preview)) > 120 {
			preview = truncateRunes(preview, 120) + "..."
		}
		if err := enc.Encode(map[string]any{
			"type":       "inspect_sample",
			"index":      i,
			"start_char": resp.Chunks[i].StartChar,
			"end_char":   resp.Chunks[i].EndChar,
			"preview":    preview,
		}); err != nil {
			return err
		}
	}
	return nil
}

func truncateRunes(s string, limit int) string {
	r := []rune(s)
	if limit < 0 || len(r) <= limit {
		return s
	}
	return string(r[:limit])
}

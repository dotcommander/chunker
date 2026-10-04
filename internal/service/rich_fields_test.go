package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dotcommander/chunker/internal/domain"
)

// Golden JSON tests pin the wire schema of the richer payload fields
// (content_hash, path, metadata). The goldens are exact json.Marshal output
// for fixed inputs; regenerating them requires reviewing the response-schema
// change (API.md documents the same shape).

const goldenMarkdownAware = `{"chunks":[{"id":0,"start_char":0,"end_char":30,"text":"# Title\n\nIntro paragraph here.","char_count":30,"word_count":5,"content_hash":"347034ca44811c26"},` +
	`{"id":1,"start_char":0,"end_char":0,"text":"` + "```go" + `\nfmt.Println(\"hi\")\n` + "```" + `","char_count":27,"word_count":3,"metadata":{"language":"go","line_count":"1"},"content_hash":"295cf8e363aeb2b5"}],` +
	`"metadata":{"total_chunks":2,"total_chars":60,"strategy_used":"markdown_aware"}}`

const goldenHeadingAware = `{"chunks":[{"id":0,"start_char":0,"end_char":18,"text":"# Alpha Alpha body","char_count":18,"word_count":4,"path":["Alpha"],"content_hash":"a30b657c56e25e0e"},` +
	`{"id":1,"start_char":19,"end_char":24,"text":"text.","char_count":5,"word_count":1,"path":["Alpha"],"content_hash":"3d7de60a233c1b63"},` +
	`{"id":2,"start_char":27,"end_char":44,"text":"## Beta Beta body","char_count":17,"word_count":4,"path":["Alpha","Beta"],"content_hash":"dc38bfbc7a0d938a"},` +
	`{"id":3,"start_char":45,"end_char":50,"text":"text.","char_count":5,"word_count":1,"path":["Alpha","Beta"],"content_hash":"3d7de60a233c1b63"}],` +
	`"metadata":{"total_chunks":4,"total_chars":52,"strategy_used":"heading_aware"}}`

func TestChunkService_ResponseSchema_MarkdownAware(t *testing.T) {
	t.Parallel()

	resp, err := NewChunkService().ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:      "# Title\n\nIntro paragraph here.\n\n```go\nfmt.Println(\"hi\")\n```\n",
		ChunkSize: 40,
		Strategy:  domain.MarkdownAware,
	})
	if err != nil {
		t.Fatalf("ProcessChunkRequest() unexpected error: %v", err)
	}

	assertGoldenJSON(t, "markdown_aware", resp, goldenMarkdownAware)
}

func TestChunkService_ResponseSchema_HeadingAware(t *testing.T) {
	t.Parallel()

	resp, err := NewChunkService().ProcessChunkRequest(context.Background(), domain.ChunkRequest{
		Text:      "# Alpha\n\nAlpha body text.\n\n## Beta\n\nBeta body text.\n",
		ChunkSize: 20,
		Strategy:  domain.HeadingAware,
	})
	if err != nil {
		t.Fatalf("ProcessChunkRequest() unexpected error: %v", err)
	}

	assertGoldenJSON(t, "heading_aware", resp, goldenHeadingAware)
}

func assertGoldenJSON(t *testing.T, name string, resp *domain.ChunkResponse, golden string) {
	t.Helper()

	actual, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("%s: marshal response: %v", name, err)
	}
	if string(actual) != golden {
		t.Errorf("%s: response JSON mismatch\n got: %s\nwant: %s", name, actual, golden)
	}
}

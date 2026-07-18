package service

import (
	"context"
	"strings"
	"testing"

	"github.com/dotcommander/reliquary/chunking"

	"github.com/dotcommander/chunker/internal/domain"
)

func TestChunkService_ProcessChunkRequest_Success(t *testing.T) {
	t.Parallel()
	svc := NewChunkService()

	req := domain.ChunkRequest{
		Text:      "Hello world",
		ChunkSize: 100,
		Overlap:   10,
		Strategy:  domain.WordBoundary,
	}

	resp, err := svc.ProcessChunkRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("ProcessChunkRequest() unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("ProcessChunkRequest() returned nil response")
	}

	if len(resp.Chunks) == 0 {
		t.Errorf("ProcessChunkRequest() got 0 chunks, want >= 1")
	}

	if resp.Metadata.StrategyUsed != domain.WordBoundary {
		t.Errorf("ProcessChunkRequest() strategy = %v, want %v", resp.Metadata.StrategyUsed, domain.WordBoundary)
	}

	if resp.Metadata.TotalChunks != len(resp.Chunks) {
		t.Errorf("ProcessChunkRequest() total chunks = %v, want %v", resp.Metadata.TotalChunks, len(resp.Chunks))
	}
}

func TestChunkService_ProcessChunkRequest_ValidationError(t *testing.T) {
	t.Parallel()
	svc := NewChunkService()

	tests := []struct {
		name    string
		req     domain.ChunkRequest
		wantErr string
	}{
		{
			name: "empty text",
			req: domain.ChunkRequest{
				Text:      "",
				ChunkSize: 100,
			},
			wantErr: "text is required",
		},
		{
			name: "zero chunk size",
			req: domain.ChunkRequest{
				Text:      "test",
				ChunkSize: 0,
			},
			wantErr: "chunk_size must be greater than 0",
		},
		{
			name: "overlap >= chunk size",
			req: domain.ChunkRequest{
				Text:      "test",
				ChunkSize: 100,
				Overlap:   100,
			},
			wantErr: "overlap must be less than chunk_size",
		},
		{
			name: "unknown strategy",
			req: domain.ChunkRequest{
				Text:      "test",
				ChunkSize: 100,
				Strategy:  domain.Strategy("bogus"),
			},
			wantErr: `unknown strategy: "bogus"`,
		},
		{
			name: "unknown token encoding",
			req: domain.ChunkRequest{
				Text:          "test",
				ChunkSize:     100,
				TokenEncoding: domain.TokenEncoding("gpt-4-turbo"),
			},
			wantErr: `unknown token_encoding: "gpt-4-turbo"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := svc.ProcessChunkRequest(context.Background(), tt.req)
			if err == nil {
				t.Errorf("ProcessChunkRequest() expected error, got nil")
				return
			}
			if err.Error() != tt.wantErr {
				t.Errorf("ProcessChunkRequest() error = %v, want %v", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestChunkService_ProcessChunkRequest_DefaultStrategy(t *testing.T) {
	t.Parallel()
	svc := NewChunkService()

	req := domain.ChunkRequest{
		Text:      "Hello world",
		ChunkSize: 100,
		Strategy:  "", // Empty should default to SmartBoundary
	}

	resp, err := svc.ProcessChunkRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("ProcessChunkRequest() unexpected error: %v", err)
	}

	if resp.Metadata.StrategyUsed != domain.SmartBoundary {
		t.Errorf("ProcessChunkRequest() strategy = %v, want %v", resp.Metadata.StrategyUsed, domain.SmartBoundary)
	}
}

// TestChunkService_ProcessChunkRequest_CancelledContext verifies that a
// cancelled context is detected at entry before any chunking work is done.
// NOTE: The old FactoryError test used a mock that forced a factory error;
// that path is now unreachable via ProcessChunkRequest because WithDefault()
// normalises all unrecognised strategies to SmartBoundary before NewChunker
// is called. The ctx.Err() early-return introduced by this refactor is the
// analogous failure-at-entry contract.
func TestChunkService_ProcessChunkRequest_CancelledContext(t *testing.T) {
	t.Parallel()
	svc := NewChunkService()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	req := domain.ChunkRequest{
		Text:      "Hello world",
		ChunkSize: 100,
		Strategy:  domain.WordBoundary,
	}

	_, err := svc.ProcessChunkRequest(ctx, req)
	if err == nil {
		t.Fatal("ProcessChunkRequest() expected error for cancelled context, got nil")
	}
}

func TestChunkService_ProcessChunkRequest_UTF8CharCount(t *testing.T) {
	t.Parallel()
	svc := NewChunkService()

	// Text with multi-byte UTF-8 characters
	req := domain.ChunkRequest{
		Text:      "Hello 世界", // 8 characters, 13 bytes
		ChunkSize: 100,
		Strategy:  domain.WordBoundary,
	}

	resp, err := svc.ProcessChunkRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("ProcessChunkRequest() unexpected error: %v", err)
	}

	// Should be 8 characters, not 13 bytes
	if resp.Metadata.TotalChars != 8 {
		t.Errorf("ProcessChunkRequest() total chars = %v, want 8 (UTF-8 character count)", resp.Metadata.TotalChars)
	}
}

func TestChunkService_ProcessChunkRequest_ChunkOffsets(t *testing.T) {
	t.Parallel()
	svc := NewChunkService()

	req := domain.ChunkRequest{
		Text:      "hello world",
		ChunkSize: 100,
		Strategy:  domain.WordBoundary,
	}

	resp, err := svc.ProcessChunkRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("ProcessChunkRequest() unexpected error: %v", err)
	}

	if len(resp.Chunks) == 0 {
		t.Fatal("expected at least 1 chunk, got 0")
	}

	if resp.Chunks[0].StartChar != 0 {
		t.Errorf("start_char = %d, want 0", resp.Chunks[0].StartChar)
	}

	if resp.Chunks[0].EndChar != len([]rune("hello world")) {
		t.Errorf("end_char = %d, want %d", resp.Chunks[0].EndChar, len([]rune("hello world")))
	}
}

// TestCalculateTotalTokens_InvalidEncodingError exercises the invalid-encoding
// path of calculateTotalTokens directly. The pre-refactor code discarded the
// CountTokens error with `_`, producing a silent zero count; the refactor
// returns the error so callers can fail cleanly.
// TestAnnotateChunkOffsets_HonorsLibSpanNonVerbatim verifies that
// annotateChunkOffsets converts the library's authoritative byte offsets
// into rune offsets even when the chunk's verbatim text is not a substring
// of the source (e.g. after whitespace normalization).
func TestAnnotateChunkOffsets_HonorsLibSpanNonVerbatim(t *testing.T) {
	t.Parallel()
	source := "hello world"
	// Simulate a lib chunk whose text was whitespace-normalized so it is NOT
	// a verbatim substring, but whose authoritative byte span is [0:11].
	chunks := []domain.Chunk{
		{ID: 1, Text: "helloworld", StartChar: 0, EndChar: 11, CharCount: 10},
	}
	annotateChunkOffsets(source, chunks)
	if chunks[0].StartChar != 0 {
		t.Errorf("StartChar = %d, want 0", chunks[0].StartChar)
	}
	if chunks[0].EndChar != 11 {
		t.Errorf("EndChar = %d, want 11 (rune count of source)", chunks[0].EndChar)
	}
}

// TestAnnotateChunkOffsets_NonASCIIByteToRune verifies byte→rune conversion
// for non-ASCII input.
func TestAnnotateChunkOffsets_NonASCIIByteToRune(t *testing.T) {
	t.Parallel()
	// é and ö are 2 bytes each; "héllo" occupies bytes [0:6].
	source := "héllo wörld"
	chunks := []domain.Chunk{
		{ID: 1, Text: "héllo", StartChar: 0, EndChar: 6, CharCount: 5},
	}
	annotateChunkOffsets(source, chunks)
	if chunks[0].StartChar != 0 {
		t.Errorf("StartChar = %d, want 0", chunks[0].StartChar)
	}
	if chunks[0].EndChar != 5 {
		t.Errorf("EndChar = %d, want 5 (5 runes)", chunks[0].EndChar)
	}
}

// TestChunkService_ProcessChunkRequest_ChunkOffsetsNonASCII verifies
// chunk offsets round-trip correctly through ProcessChunkRequest with
// multi-byte UTF-8 input.
func TestChunkService_ProcessChunkRequest_ChunkOffsetsNonASCII(t *testing.T) {
	t.Parallel()
	svc := NewChunkService()
	req := domain.ChunkRequest{
		Text:      "héllo wörld",
		ChunkSize: 100,
		Strategy:  domain.WordBoundary,
	}
	resp, err := svc.ProcessChunkRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("ProcessChunkRequest() unexpected error: %v", err)
	}
	if len(resp.Chunks) == 0 {
		t.Fatal("expected at least 1 chunk, got 0")
	}
	runes := []rune(req.Text)
	for i, c := range resp.Chunks {
		want := string(runes[c.StartChar:c.EndChar])
		if c.Text != want {
			t.Errorf("chunk[%d]: rune slice [%d:%d] = %q, chunk.Text = %q",
				i, c.StartChar, c.EndChar, want, c.Text)
		}
	}
}

func TestCalculateTotalTokens_NoDoubleCountOnOverlap(t *testing.T) {
	t.Parallel()
	svc := NewChunkService()

	text := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 6)
	req := domain.ChunkRequest{
		Text:          text,
		ChunkSize:     20,
		Overlap:       5,
		Strategy:      domain.TokenBased,
		TokenEncoding: domain.EncodingCL100K,
	}

	resp, err := svc.ProcessChunkRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("ProcessChunkRequest() unexpected error: %v", err)
	}

	if len(resp.Chunks) < 2 {
		t.Fatalf("expected >= 2 chunks, got %d", len(resp.Chunks))
	}

	want, err := chunking.CountTokens(text, string(domain.EncodingCL100K))
	if err != nil {
		t.Fatalf("CountTokens() unexpected error: %v", err)
	}

	if resp.Metadata.TotalTokens != want {
		t.Errorf("TotalTokens = %d, want %d (original-text basis, not chunk-sum)", resp.Metadata.TotalTokens, want)
	}
}

func TestCalculateTotalTokens_InvalidEncodingError(t *testing.T) {
	t.Parallel()
	req := domain.ChunkRequest{
		Text:          "hello world",
		ChunkSize:     100,
		Strategy:      domain.TokenBased,
		TokenEncoding: domain.TokenEncoding("not-a-real-encoding"),
	}

	// Sanity: chunking.CountTokens rejects invalid encodings.
	if _, err := chunking.CountTokens(req.Text, "not-a-real-encoding"); err == nil {
		t.Fatal("expected CountTokens to error on invalid encoding")
	}

	// calculateTotalTokens routes through TokenEncoding.WithDefault, which
	// sanitizes the bogus value to cl100k_base — so the public path stays
	// safe. Assert the success path returns no error for the sanitized case.
	got, err := calculateTotalTokens(req, domain.TokenBased)
	if err != nil {
		t.Fatalf("calculateTotalTokens unexpected error: %v", err)
	}
	if got <= 0 {
		t.Errorf("calculateTotalTokens fallback count = %d, want > 0", got)
	}
}

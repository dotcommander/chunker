package domain

import "testing"

func TestBatchChunkRequest_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		req     BatchChunkRequest
		wantErr string
	}{
		{
			name:    "missing texts",
			req:     BatchChunkRequest{ChunkSize: 100},
			wantErr: "texts must contain at least one text",
		},
		{
			name:    "bad chunk size",
			req:     BatchChunkRequest{Texts: []string{"a"}, ChunkSize: 0},
			wantErr: "chunk_size must be greater than 0",
		},
		{
			name:    "bad strategy",
			req:     BatchChunkRequest{Texts: []string{"a"}, ChunkSize: 100, Strategy: "bogus"},
			wantErr: "unknown strategy",
		},
		{
			name:    "overlap not less than chunk size",
			req:     BatchChunkRequest{Texts: []string{"a"}, ChunkSize: 100, Overlap: 100},
			wantErr: "overlap must be less than chunk_size",
		},
		{
			name: "empty text is not a batch-level error",
			req:  BatchChunkRequest{Texts: []string{"a", ""}, ChunkSize: 100},
		},
		{
			name: "valid",
			req:  BatchChunkRequest{Texts: []string{"a", "b"}, ChunkSize: 100, Strategy: Recursive, SeparatorProfile: ProfileCJKThai, MaxTokens: 512},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.req.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() unexpected error: %v", err)
				}
				return
			}
			if !IsRequestValidationError(err) {
				t.Fatalf("Validate() error = %v, want a request validation error", err)
			}
			if got := err.Error(); len(got) < len(tt.wantErr) || got[:len(tt.wantErr)] != tt.wantErr {
				t.Errorf("Validate() error = %q, want prefix %q", got, tt.wantErr)
			}
		})
	}
}

func TestBatchChunkRequest_ToChunkRequest(t *testing.T) {
	t.Parallel()

	batch := BatchChunkRequest{
		Texts:             []string{"ignored at conversion"},
		ChunkSize:         250,
		Strategy:          Semantic,
		Overlap:           10,
		TokenEncoding:     EncodingO200K,
		SeparatorProfile:  ProfileCJKThai,
		Semantic:          &SemanticOptions{BreakSensitivity: 1.5},
		MaxTokens:         128,
		IncludeEmbeddings: true,
	}

	got := batch.ToChunkRequest("the text")
	if got.Text != "the text" {
		t.Errorf("Text = %q, want the per-item text", got.Text)
	}
	if got.ChunkSize != batch.ChunkSize || got.Strategy != batch.Strategy || got.Overlap != batch.Overlap {
		t.Errorf("basic params not carried: %+v", got)
	}
	if got.TokenEncoding != EncodingO200K || got.SeparatorProfile != ProfileCJKThai || got.MaxTokens != 128 || !got.IncludeEmbeddings {
		t.Errorf("extended params not carried: %+v", got)
	}
	if got.Semantic == nil || got.Semantic.BreakSensitivity != 1.5 {
		t.Errorf("Semantic not carried: %+v", got.Semantic)
	}
}

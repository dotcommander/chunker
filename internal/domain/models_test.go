package domain

import (
	"testing"
)

func TestChunkRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     ChunkRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			req: ChunkRequest{
				Text:      "Hello world",
				ChunkSize: 100,
				Overlap:   10,
			},
			wantErr: false,
		},
		{
			name: "empty text",
			req: ChunkRequest{
				Text:      "",
				ChunkSize: 100,
			},
			wantErr: true,
			errMsg:  "text is required",
		},
		{
			name: "zero chunk size",
			req: ChunkRequest{
				Text:      "Hello",
				ChunkSize: 0,
			},
			wantErr: true,
			errMsg:  "chunk_size must be greater than 0",
		},
		{
			name: "negative chunk size",
			req: ChunkRequest{
				Text:      "Hello",
				ChunkSize: -1,
			},
			wantErr: true,
			errMsg:  "chunk_size must be greater than 0",
		},
		{
			name: "negative overlap",
			req: ChunkRequest{
				Text:      "Hello",
				ChunkSize: 100,
				Overlap:   -1,
			},
			wantErr: true,
			errMsg:  "overlap must be greater than or equal to 0",
		},
		{
			name: "overlap equals chunk size",
			req: ChunkRequest{
				Text:      "Hello",
				ChunkSize: 100,
				Overlap:   100,
			},
			wantErr: true,
			errMsg:  "overlap must be less than chunk_size",
		},
		{
			name: "overlap greater than chunk size",
			req: ChunkRequest{
				Text:      "Hello",
				ChunkSize: 100,
				Overlap:   150,
			},
			wantErr: true,
			errMsg:  "overlap must be less than chunk_size",
		},
		{
			name: "valid with zero overlap",
			req: ChunkRequest{
				Text:      "Hello",
				ChunkSize: 100,
				Overlap:   0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error, got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestStrategy_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		strategy Strategy
		want     bool
	}{
		{"word_boundary valid", WordBoundary, true},
		{"sentence_boundary valid", SentenceBoundary, true},
		{"hard_cut valid", HardCut, true},
		{"paragraph_aware valid", ParagraphAware, true},
		{"token_based valid", TokenBased, true},
		{"smart_boundary valid", SmartBoundary, true},
		{"markdown_aware valid", MarkdownAware, true},
		{"empty invalid", Strategy(""), false},
		{"invalid strategy", Strategy("invalid"), false},
		{"uppercase invalid", Strategy("WORD_BOUNDARY"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.strategy.IsValid(); got != tt.want {
				t.Errorf("Strategy.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStrategy_WithDefault(t *testing.T) {
	tests := []struct {
		name     string
		strategy Strategy
		want     Strategy
	}{
		{"valid returns same", WordBoundary, WordBoundary},
		{"empty returns default", Strategy(""), SmartBoundary},
		{"invalid returns default", Strategy("invalid"), SmartBoundary},
		{"smart_boundary returns same", SmartBoundary, SmartBoundary},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.strategy.WithDefault(); got != tt.want {
				t.Errorf("Strategy.WithDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenEncoding_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		encoding TokenEncoding
		want     bool
	}{
		{"cl100k_base valid", EncodingCL100K, true},
		{"o200k_base valid", EncodingO200K, true},
		{"p50k_base valid", EncodingP50K, true},
		{"r50k_base valid", EncodingR50K, true},
		{"empty invalid", TokenEncoding(""), false},
		{"invalid encoding", TokenEncoding("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.encoding.IsValid(); got != tt.want {
				t.Errorf("TokenEncoding.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTokenEncoding_WithDefault(t *testing.T) {
	tests := []struct {
		name     string
		encoding TokenEncoding
		want     TokenEncoding
	}{
		{"valid returns same", EncodingO200K, EncodingO200K},
		{"empty returns default", TokenEncoding(""), EncodingCL100K},
		{"invalid returns default", TokenEncoding("invalid"), EncodingCL100K},
		{"cl100k returns same", EncodingCL100K, EncodingCL100K},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.encoding.WithDefault(); got != tt.want {
				t.Errorf("TokenEncoding.WithDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

package domain

import (
	"errors"
	"fmt"
)

type requestValidationError struct {
	message string
}

func (e *requestValidationError) Error() string {
	return e.message
}

func invalidRequest(format string, args ...any) error {
	return &requestValidationError{message: fmt.Sprintf(format, args...)}
}

// IsRequestValidationError reports whether err came from ChunkRequest.Validate.
// Handlers use this boundary to distinguish safe client errors from internal
// service failures without coupling status codes to error-message text.
func IsRequestValidationError(err error) bool {
	var validationErr *requestValidationError
	return errors.As(err, &validationErr)
}

type ChunkRequest struct {
	Text          string        `json:"text"`
	ChunkSize     int           `json:"chunk_size"`
	Strategy      Strategy      `json:"strategy,omitempty"`
	Overlap       int           `json:"overlap,omitempty"`
	TokenEncoding TokenEncoding `json:"token_encoding,omitempty"`
	// MaxTokens caps every chunk's token count (measured with TokenEncoding,
	// default cl100k_base) after the base strategy runs. 0 disables the cap.
	// Chunks that exceed it are split further (sentence, then word, then
	// hard cut), so strategies can stay semantic while honoring LLM context
	// limits. It is independent of chunk_size: chunk_size bounds characters
	// (tokens for token_based); max_tokens bounds tokens on every strategy.
	MaxTokens int `json:"max_tokens,omitempty"`
}

// Validate performs business logic validation.
func (r ChunkRequest) Validate() error {
	// Required fields
	if r.Text == "" {
		return invalidRequest("text is required")
	}

	if r.ChunkSize <= 0 {
		return invalidRequest("chunk_size must be greater than 0")
	}

	if r.Overlap < 0 {
		return invalidRequest("overlap must be greater than or equal to 0")
	}

	// Custom validation for overlap vs chunk size
	if r.Overlap >= r.ChunkSize {
		return invalidRequest("overlap must be less than chunk_size")
	}

	if r.Strategy != "" && !r.Strategy.IsValid() {
		return invalidRequest("unknown strategy: %q", r.Strategy)
	}

	if r.TokenEncoding != "" && !r.TokenEncoding.IsValid() {
		return invalidRequest("unknown token_encoding: %q", r.TokenEncoding)
	}

	if r.MaxTokens < 0 {
		return invalidRequest("max_tokens must be greater than or equal to 0")
	}

	return nil
}

type ChunkResponse struct {
	Chunks   []Chunk  `json:"chunks"`
	Metadata Metadata `json:"metadata"`
}

type Chunk struct {
	ID         int    `json:"id"`
	StartChar  int    `json:"start_char"`
	EndChar    int    `json:"end_char"`
	Text       string `json:"text"`
	CharCount  int    `json:"char_count"`
	WordCount  int    `json:"word_count"`
	TokenCount int    `json:"token_count,omitempty"`
	// Path is the heading breadcrumb ("H1 Title" → "Subsection") for
	// heading-aware strategies; omitted otherwise.
	Path []string `json:"path,omitempty"`
	// Metadata carries strategy-produced annotations (for example
	// markdown_aware's heading_level, language, line_count); omitted when the
	// strategy produces none.
	Metadata map[string]string `json:"metadata,omitempty"`
	// ContentHash is the first 16 hex characters of SHA-256(Text).
	ContentHash string `json:"content_hash"`
}

type Metadata struct {
	TotalChunks   int           `json:"total_chunks"`
	TotalChars    int           `json:"total_chars"`
	TotalTokens   int           `json:"total_tokens,omitempty"`
	StrategyUsed  Strategy      `json:"strategy_used"`
	TokenEncoding TokenEncoding `json:"token_encoding,omitempty"`
	// MaxTokens reports the token ceiling that was enforced, set only when
	// the request asked for one.
	MaxTokens int `json:"max_tokens,omitempty"`
}

type Strategy string

const (
	WordBoundary     Strategy = "word_boundary"
	SentenceBoundary Strategy = "sentence_boundary"
	HardCut          Strategy = "hard_cut"
	ParagraphAware   Strategy = "paragraph_aware"
	TokenBased       Strategy = "token_based"
	SmartBoundary    Strategy = "smart_boundary"
	MarkdownAware    Strategy = "markdown_aware"
	HeadingAware     Strategy = "heading_aware"
)

func (s Strategy) IsValid() bool {
	switch s {
	case WordBoundary, SentenceBoundary, HardCut, ParagraphAware, TokenBased, SmartBoundary, MarkdownAware, HeadingAware:
		return true
	}
	return false
}

func (s Strategy) WithDefault() Strategy {
	if s == "" || !s.IsValid() {
		return SmartBoundary
	}
	return s
}

type TokenEncoding string

const (
	EncodingCL100K TokenEncoding = "cl100k_base" // GPT-3.5/GPT-4
	EncodingO200K  TokenEncoding = "o200k_base"  // gpt-5
	EncodingP50K   TokenEncoding = "p50k_base"   // Older models
	EncodingR50K   TokenEncoding = "r50k_base"   // Very old models
)

func (e TokenEncoding) IsValid() bool {
	switch e {
	case EncodingCL100K, EncodingO200K, EncodingP50K, EncodingR50K:
		return true
	}
	return false
}

func (e TokenEncoding) WithDefault() TokenEncoding {
	if e == "" || !e.IsValid() {
		return EncodingCL100K
	}
	return e
}

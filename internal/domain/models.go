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
	// SeparatorProfile selects the recursive strategy's separator cascade:
	// "default_text" (empty) or "cjk_thai". Only valid with strategy=recursive.
	SeparatorProfile SeparatorProfile `json:"separator_profile,omitempty"`
	// Semantic tunes the semantic strategy; zero fields fall back to library
	// defaults. Only valid with strategy=semantic.
	Semantic *SemanticOptions `json:"semantic,omitempty"`
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

	if r.SeparatorProfile != "" {
		if !r.SeparatorProfile.IsValid() {
			return invalidRequest("unknown separator_profile: %q", r.SeparatorProfile)
		}
		if r.Strategy != Recursive {
			return invalidRequest("separator_profile requires the recursive strategy")
		}
	}

	if r.Semantic != nil {
		if r.Strategy != Semantic {
			return invalidRequest("semantic options require the semantic strategy")
		}
		if err := r.Semantic.validate(); err != nil {
			return err
		}
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
	// SeparatorProfile reports the recursive separator cascade in effect,
	// set only when the request specified one.
	SeparatorProfile SeparatorProfile `json:"separator_profile,omitempty"`
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
	Recursive        Strategy = "recursive"
	Semantic         Strategy = "semantic"
)

func (s Strategy) IsValid() bool {
	switch s {
	case WordBoundary, SentenceBoundary, HardCut, ParagraphAware, TokenBased, SmartBoundary, MarkdownAware, HeadingAware, Recursive, Semantic:
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

// SemanticUnavailable returns the request-validation error reported when the
// semantic strategy is requested while no embedding configuration exists.
// The semantic strategy needs an embedder; until the embedding seam is
// wired, explicitly requesting it must fail clearly, not silently fall back.
func SemanticUnavailable() error {
	return invalidRequest("semantic strategy requires embedding configuration")
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

type SeparatorProfile string

// SemanticOptions tunes the semantic strategy. Zero values mean library
// defaults (see reliquary chunking.SemanticOpts): MaxChunkChars 1600,
// MinChunkChars 200, BreakSensitivity 1.0; smoothing and coherence windows
// are disabled by default.
type SemanticOptions struct {
	BreakSensitivity float64 `json:"break_sensitivity,omitempty"`
	MinChunkChars    int     `json:"min_chunk_chars,omitempty"`
	MaxChunkChars    int     `json:"max_chunk_chars,omitempty"`
	SmoothingWindow  int     `json:"smoothing_window,omitempty"`
	CoherenceWindow  int     `json:"coherence_window,omitempty"`
}

func (o *SemanticOptions) validate() error {
	if o.BreakSensitivity < 0 {
		return invalidRequest("semantic.break_sensitivity must be greater than or equal to 0")
	}
	if o.MinChunkChars < 0 || o.MaxChunkChars < 0 {
		return invalidRequest("semantic chunk char bounds must be greater than or equal to 0")
	}
	if o.MinChunkChars > 0 && o.MaxChunkChars > 0 && o.MaxChunkChars < o.MinChunkChars {
		return invalidRequest("semantic.max_chunk_chars must be greater than or equal to semantic.min_chunk_chars")
	}
	if o.SmoothingWindow < 0 || o.CoherenceWindow < 0 {
		return invalidRequest("semantic windows must be greater than or equal to 0")
	}
	return nil
}

const (
	ProfileDefaultText SeparatorProfile = "default_text"
	ProfileCJKThai     SeparatorProfile = "cjk_thai"
)

func (p SeparatorProfile) IsValid() bool {
	switch p {
	case ProfileDefaultText, ProfileCJKThai:
		return true
	}
	return false
}

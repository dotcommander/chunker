package domain

type ChunkRequest struct {
	Text          string        `json:"text"`
	ChunkSize     int           `json:"chunk_size"`
	Strategy      Strategy      `json:"strategy,omitempty"`
	Overlap       int           `json:"overlap,omitempty"`
	TokenEncoding TokenEncoding `json:"token_encoding,omitempty"`
}

type ChunkResponse struct {
	Chunks   []Chunk  `json:"chunks"`
	Metadata Metadata `json:"metadata"`
}

type Chunk struct {
	ID         int    `json:"id"`
	Text       string `json:"text"`
	CharCount  int    `json:"char_count"`
	WordCount  int    `json:"word_count"`
	TokenCount int    `json:"token_count,omitempty"`
}

type Metadata struct {
	TotalChunks    int           `json:"total_chunks"`
	TotalChars     int           `json:"total_chars"`
	TotalTokens    int           `json:"total_tokens,omitempty"`
	StrategyUsed   Strategy      `json:"strategy_used"`
	TokenEncoding  TokenEncoding `json:"token_encoding,omitempty"`
}

type Strategy string

const (
	WordBoundary      Strategy = "word_boundary"
	SentenceBoundary  Strategy = "sentence_boundary"
	HardCut           Strategy = "hard_cut"
	ParagraphAware    Strategy = "paragraph_aware"
	TokenBased        Strategy = "token_based"
	SmartBoundary     Strategy = "smart_boundary"
)

func (s Strategy) IsValid() bool {
	switch s {
	case WordBoundary, SentenceBoundary, HardCut, ParagraphAware, TokenBased, SmartBoundary:
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
	EncodingCL100K  TokenEncoding = "cl100k_base"  // GPT-3.5, GPT-4
	EncodingO200K   TokenEncoding = "o200k_base"   // GPT-4o
	EncodingP50K    TokenEncoding = "p50k_base"    // Older models
	EncodingR50K    TokenEncoding = "r50k_base"    // Very old models
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
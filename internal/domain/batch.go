package domain

// BatchChunkRequest chunks many texts under one shared parameter set. Each
// text is processed independently: item-level failures (for example an empty
// text) surface as per-item error results, while batch-level problems (bad
// shared params, missing texts) fail the whole request.
type BatchChunkRequest struct {
	Texts             []string         `json:"texts"`
	ChunkSize         int              `json:"chunk_size"`
	Strategy          Strategy         `json:"strategy,omitempty"`
	Overlap           int              `json:"overlap,omitempty"`
	TokenEncoding     TokenEncoding    `json:"token_encoding,omitempty"`
	SeparatorProfile  SeparatorProfile `json:"separator_profile,omitempty"`
	Semantic          *SemanticOptions `json:"semantic,omitempty"`
	MaxTokens         int              `json:"max_tokens,omitempty"`
	IncludeEmbeddings bool             `json:"include_embeddings,omitempty"`
}

// ToChunkRequest converts the shared parameters plus one text into a single
// chunk request — the exact transformation batch processing applies per item.
func (r BatchChunkRequest) ToChunkRequest(text string) ChunkRequest {
	return ChunkRequest{
		Text:              text,
		ChunkSize:         r.ChunkSize,
		Strategy:          r.Strategy,
		Overlap:           r.Overlap,
		TokenEncoding:     r.TokenEncoding,
		SeparatorProfile:  r.SeparatorProfile,
		Semantic:          r.Semantic,
		MaxTokens:         r.MaxTokens,
		IncludeEmbeddings: r.IncludeEmbeddings,
	}
}

// Validate checks the batch shape and the shared parameters once. The probe
// text is non-empty so only shared fields are exercised; per-text validity
// (empty text) is an item-level result, not a whole-batch failure.
func (r BatchChunkRequest) Validate() error {
	if len(r.Texts) == 0 {
		return invalidRequest("texts must contain at least one text")
	}
	return r.ToChunkRequest("probe text").Validate()
}

// BatchChunkResult carries exactly one of Response or Error for one input
// text, keyed by its index in the request — no partial-failure ambiguity.
type BatchChunkResult struct {
	Index    int            `json:"index"`
	Response *ChunkResponse `json:"response,omitempty"`
	Error    string         `json:"error,omitempty"`
}

type BatchChunkResponse struct {
	Results []BatchChunkResult `json:"results"`
}

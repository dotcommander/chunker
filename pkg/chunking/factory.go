package chunking

import (
	"fmt"

	"chunker/internal/domain"
)

type Factory struct{}

func NewFactory() *Factory {
	return &Factory{}
}

func (f *Factory) CreateChunker(strategy domain.Strategy) (domain.Chunker, error) {
	switch strategy {
	case domain.WordBoundary:
		return NewWordBoundaryChunker(), nil
	case domain.SentenceBoundary:
		return NewSentenceBoundaryChunker(), nil
	case domain.HardCut:
		return NewHardCutChunker(), nil
	case domain.ParagraphAware:
		return NewParagraphAwareChunker(), nil
	case domain.TokenBased:
		return NewTokenBasedChunker(), nil
	case domain.SmartBoundary:
		return NewSmartBoundaryChunker(), nil
	default:
		return nil, fmt.Errorf("unknown strategy: %s", strategy)
	}
}
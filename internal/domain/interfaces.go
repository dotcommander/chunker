package domain

import "context"

type Chunker interface {
	Chunk(ctx context.Context, text string, size int, overlap int) []Chunk
	Strategy() Strategy
}

type TokenChunker interface {
	Chunker
	ChunkWithEncoding(ctx context.Context, text string, maxTokens int, overlap int, encoding string) []Chunk
}

type ChunkerFactory interface {
	CreateChunker(strategy Strategy) (Chunker, error)
}

type ChunkService interface {
	ProcessChunkRequest(ctx context.Context, req ChunkRequest) (*ChunkResponse, error)
}
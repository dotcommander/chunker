package domain

import "context"

type ChunkService interface {
	ProcessChunkRequest(ctx context.Context, req ChunkRequest) (*ChunkResponse, error)
}

package embeddings

import (
	"context"
	
	"github.com/michaelgalloway/sophia/internal/datasources"
)

// Vector represents an embedding vector
type Vector []float32

// EmbeddingService handles the creation and management of embeddings
type EmbeddingService interface {
	// CreateEmbedding generates an embedding vector for the given text
	CreateEmbedding(ctx context.Context, text string) (Vector, error)
	
	// CreateEmbeddings generates embedding vectors for multiple documents
	CreateEmbeddings(ctx context.Context, docs []datasources.Document) ([]Vector, error)
	
	// QueryEmbedding generates an embedding vector for a query
	QueryEmbedding(ctx context.Context, query string) (Vector, error)
}

// Config holds configuration for the embedding service
type Config struct {
	OpenAIKey     string
	ModelName     string
	BatchSize     int
	MaxRetries    int
	RetryInterval int
}

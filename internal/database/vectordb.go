package database

import (
	"context"

	"github.com/michaelgalloway/sophia/internal/datasources"
	"github.com/michaelgalloway/sophia/internal/embeddings"
)

// SearchResult represents a single search result with its similarity score
type SearchResult struct {
	Document datasources.Document
	Score    float64
}

// VectorDB defines the interface for vector database operations
type VectorDB interface {
	// Store saves documents and their embeddings
	Store(ctx context.Context, docs []datasources.Document, vectors []embeddings.Vector) error

	// Search finds similar documents based on a query vector
	Search(ctx context.Context, queryVector embeddings.Vector, limit int) ([]SearchResult, error)

	// DeleteBySource removes all documents from a specific source
	DeleteBySource(ctx context.Context, source string) error

	// DeleteAll removes all documents
	DeleteAll(ctx context.Context) error

	// Initialize sets up the database connection and schema
	Initialize(ctx context.Context) error
}

// Config holds configuration for the vector database
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

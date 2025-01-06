package datasources

import (
	"context"
	"time"
)

// Document represents a piece of content from any data source
type Document struct {
	ID        string
	Content   string
	Title     string
	URL       string
	Metadata  map[string]interface{}
	Source    string
	Timestamp time.Time
}

// DataSource defines the interface that all data sources must implement
type DataSource interface {
	// Name returns the unique identifier for this data source
	Name() string

	// FetchData retrieves new data since the last sync
	FetchData(ctx context.Context, since time.Time) ([]Document, error)

	// Initialize sets up any necessary connections or authentication
	Initialize(ctx context.Context) error
}

// SourceFactory is a function type that creates new DataSource instances
type SourceFactory func(config map[string]interface{}) (DataSource, error)

// MaxContentLength is the maximum number of characters we'll allow for document content
// This is a conservative estimate to stay within OpenAI's 8192 token limit
const MaxContentLength = 6000

// TruncateContent ensures content stays within reasonable token limits
func TruncateContent(content string) string {
	if len(content) <= MaxContentLength {
		return content
	}
	return content[:MaxContentLength] + "... (truncated)"
}

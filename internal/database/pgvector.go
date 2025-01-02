package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	
	_ "github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	
	"github.com/michaelgalloway/sophia/internal/datasources"
	"github.com/michaelgalloway/sophia/internal/embeddings"
)

type PGVectorDB struct {
	db *sql.DB
}

func NewPGVectorDB(config Config) (*PGVectorDB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host,
		config.Port,
		config.User,
		config.Password,
		config.DBName,
		config.SSLMode,
	)
	
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	return &PGVectorDB{db: db}, nil
}

func (p *PGVectorDB) Initialize(ctx context.Context) error {
	// Create the vector extension if it doesn't exist
	_, err := p.db.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}
	
	// Create the documents table
	_, err = p.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			content TEXT NOT NULL,
			metadata JSONB,
			source TEXT NOT NULL,
			timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			embedding vector(1536)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create documents table: %w", err)
	}
	
	// Create an index for faster similarity search
	_, err = p.db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS documents_embedding_idx ON documents 
		USING ivfflat (embedding vector_cosine_ops)
		WITH (lists = 100)
	`)
	if err != nil {
		return fmt.Errorf("failed to create embedding index: %w", err)
	}
	
	return nil
}

func (p *PGVectorDB) Store(ctx context.Context, docs []datasources.Document, vectors []embeddings.Vector) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO documents (id, content, metadata, source, timestamp, embedding)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			content = EXCLUDED.content,
			metadata = EXCLUDED.metadata,
			timestamp = EXCLUDED.timestamp,
			embedding = EXCLUDED.embedding
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	
	for i, doc := range docs {
		metadata, err := json.Marshal(doc.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		
		vec := pgvector.NewVector(vectors[i])
		
		_, err = stmt.ExecContext(ctx, doc.ID, doc.Content, metadata, doc.Source, doc.Timestamp, vec)
		if err != nil {
			return fmt.Errorf("failed to insert document: %w", err)
		}
	}
	
	return tx.Commit()
}

func (p *PGVectorDB) Search(ctx context.Context, queryVector embeddings.Vector, limit int) ([]SearchResult, error) {
	vec := pgvector.NewVector(queryVector)
	
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, content, metadata, source, timestamp, 
			1 - (embedding <=> $1) as similarity
		FROM documents
		ORDER BY embedding <=> $1
		LIMIT $2
	`, vec, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()
	
	var results []SearchResult
	for rows.Next() {
		var doc datasources.Document
		var metadataJSON []byte
		var similarity float64
		
		err := rows.Scan(&doc.ID, &doc.Content, &metadataJSON, &doc.Source, &doc.Timestamp, &similarity)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		
		err = json.Unmarshal(metadataJSON, &doc.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
		
		results = append(results, SearchResult{
			Document: doc,
			Score:    similarity,
		})
	}
	
	return results, nil
}

func (p *PGVectorDB) DeleteBySource(ctx context.Context, source string) error {
	_, err := p.db.ExecContext(ctx, "DELETE FROM documents WHERE source = $1", source)
	return err
}

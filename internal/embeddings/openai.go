package embeddings

import (
	"context"
	"fmt"
	
	"github.com/sashabaranov/go-openai"
	"github.com/michaelgalloway/sophia/internal/datasources"
)

type OpenAIEmbedding struct {
	client    *openai.Client
	modelName string
	config    Config
}

func NewOpenAIEmbedding(config Config) *OpenAIEmbedding {
	client := openai.NewClient(config.OpenAIKey)
	return &OpenAIEmbedding{
		client:    client,
		modelName: config.ModelName,
		config:    config,
	}
}

func (o *OpenAIEmbedding) CreateEmbedding(ctx context.Context, text string) (Vector, error) {
	resp, err := o.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.AdaEmbeddingV2,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}
	
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data received")
	}
	
	// Convert []float32 to our Vector type
	embedding := make(Vector, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		embedding[i] = float32(v)
	}
	
	return embedding, nil
}

func (o *OpenAIEmbedding) CreateEmbeddings(ctx context.Context, docs []datasources.Document) ([]Vector, error) {
	var texts []string
	for _, doc := range docs {
		texts = append(texts, doc.Content)
	}
	
	// Process in batches to respect API limits
	batchSize := o.config.BatchSize
	if batchSize == 0 {
		batchSize = 100
	}
	
	var allEmbeddings []Vector
	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		
		batch := texts[i:end]
		resp, err := o.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
			Input: batch,
			Model: openai.AdaEmbeddingV2,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create embeddings batch %d: %w", i/batchSize, err)
		}
		
		for _, data := range resp.Data {
			embedding := make(Vector, len(data.Embedding))
			for j, v := range data.Embedding {
				embedding[j] = float32(v)
			}
			allEmbeddings = append(allEmbeddings, embedding)
		}
	}
	
	return allEmbeddings, nil
}

func (o *OpenAIEmbedding) QueryEmbedding(ctx context.Context, query string) (Vector, error) {
	return o.CreateEmbedding(ctx, query)
}

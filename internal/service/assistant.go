package service

import (
	"context"
	"fmt"
	"log"

	"github.com/michaelgalloway/sophia/internal/database"
	"github.com/michaelgalloway/sophia/internal/embeddings"
	"github.com/sashabaranov/go-openai"
)

// Assistant provides the main service functionality
type Assistant struct {
	openAIClient     *openai.Client
	embeddingService embeddings.EmbeddingService
	vectorDB         database.VectorDB
}

// Config holds the configuration for the Assistant service
type Config struct {
	OpenAIKey string
	ModelName string
}

// NewAssistant creates a new instance of the Assistant service
func NewAssistant(
	config Config,
	embeddingService embeddings.EmbeddingService,
	vectorDB database.VectorDB,
) *Assistant {
	client := openai.NewClient(config.OpenAIKey)

	return &Assistant{
		openAIClient:     client,
		embeddingService: embeddingService,
		vectorDB:         vectorDB,
	}
}

// Ask processes a user query and returns a response
func (a *Assistant) Ask(ctx context.Context, query string) (string, error) {
	// Generate embedding for the query
	queryVector, err := a.embeddingService.QueryEmbedding(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to create query embedding: %w", err)
	}

	// Search for relevant documents
	results, err := a.vectorDB.Search(ctx, queryVector, 25)
	if err != nil {
		return "", fmt.Errorf("failed to search vector database: %w", err)
	}

	// Construct prompt with context
	prompt := constructPrompt(query, results)

	// Generate response using OpenAI
	resp, err := a.openAIClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4oMini20240718,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a helpful assistant with access to the user's personal information. Use the context provided to give accurate and relevant answers.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
		},
	)
	if err != nil {
		log.Printf("failed to generate response: %w", err)
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	return resp.Choices[0].Message.Content, nil
}

func constructPrompt(query string, results []database.SearchResult) string {
	prompt := fmt.Sprintf("Question: %s\n\nRelevant Context:\n", query)

	for i, result := range results {
		prompt += fmt.Sprintf("\n%d. From %s (%s):\n%s\n",
			i+1,
			result.Document.Source,
			result.Document.Timestamp.Format("2006-01-02 15:04:05"),
			result.Document.Content,
		)
	}

	prompt += "\nPlease provide a response based on the above context."
	return prompt
}

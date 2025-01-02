package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/rs/cors"

	"github.com/michaelgalloway/sophia/internal/config"
	"github.com/michaelgalloway/sophia/internal/database"
	"github.com/michaelgalloway/sophia/internal/datasources"
	"github.com/michaelgalloway/sophia/internal/datasources/gcalendar"
	"github.com/michaelgalloway/sophia/internal/datasources/gdocs"
	"github.com/michaelgalloway/sophia/internal/datasources/gmail"
	"github.com/michaelgalloway/sophia/internal/datasources/slack"
	"github.com/michaelgalloway/sophia/internal/embeddings"
	"github.com/michaelgalloway/sophia/internal/scheduler"
	"github.com/michaelgalloway/sophia/internal/service"
)

func initializeSources(ctx context.Context, sourceConfig config.DataSourceConfig, tokenDir string, googleCreds []byte) (map[string]datasources.DataSource, error) {
	sources := make(map[string]datasources.DataSource)

	if sourceConfig.GoogleCalendar {
		calendarConfig := map[string]interface{}{
			"credentials": string(googleCreds),
			"token_dir":   tokenDir,
		}

		calendarSource, err := gcalendar.New(calendarConfig)
		if err != nil {
			return nil, err
		}
		if err := calendarSource.Initialize(ctx); err != nil {
			return nil, err
		}
		sources[calendarSource.Name()] = calendarSource
	}

	if sourceConfig.Gmail {
		gmailConfig := map[string]interface{}{
			"credentials": string(googleCreds),
			"token_dir":   tokenDir,
		}

		gmailSource, err := gmail.New(gmailConfig)
		if err != nil {
			return nil, err
		}
		if err := gmailSource.Initialize(ctx); err != nil {
			return nil, err
		}
		sources[gmailSource.Name()] = gmailSource
	}

	if sourceConfig.GoogleDocs {
		docsConfig := map[string]interface{}{
			"credentials": string(googleCreds),
			"token_dir":   tokenDir,
		}

		docsSource, err := gdocs.New(docsConfig)
		if err != nil {
			return nil, err
		}
		if err := docsSource.Initialize(ctx); err != nil {
			return nil, err
		}
		sources[docsSource.Name()] = docsSource
	}

	if sourceConfig.Slack {
		slackConfig := map[string]interface{}{
			"token": os.Getenv("SLACK_TOKEN"),
		}

		slackSource, err := slack.New(slackConfig)
		if err != nil {
			return nil, err
		}
		if err := slackSource.Initialize(ctx); err != nil {
			return nil, err
		}
		sources[slackSource.Name()] = slackSource
	}

	return sources, nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Read Google credentials file
	googleCredsPath := os.Getenv("GOOGLE_CREDENTIALS")
	googleCreds, err := os.ReadFile(googleCredsPath)
	if err != nil {
		log.Fatalf("Failed to read Google credentials file: %v", err)
	}

	ctx := context.Background()

	// Configure which data sources to enable
	sourceConfig := config.DataSourceConfig{
		GoogleCalendar: true,  // Enable Google Calendar
		Gmail:          false, // Disable Gmail for now
		GoogleDocs:     true,  // Disable Google Docs for now
		Slack:          false, // Disable Slack for now
	}

	// Initialize data sources
	sources, err := initializeSources(ctx, sourceConfig, "./tokens", googleCreds)
	if err != nil {
		log.Fatalf("Failed to initialize data sources: %v", err)
	}

	// Initialize embedding service
	embeddingService := embeddings.NewOpenAIEmbedding(embeddings.Config{
		OpenAIKey: os.Getenv("OPENAI_API_KEY"),
		ModelName: "text-embedding-ada-002",
		BatchSize: 100,
	})

	cfg := database.Config{
		Host:     os.Getenv("POSTGRES_HOST"),
		Port:     5432,
		User:     os.Getenv("POSTGRES_USER"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		DBName:   os.Getenv("POSTGRES_DB"),
		SSLMode:  "disable",
	}

	// Initialize vector database
	vectorDB, err := database.NewPGVectorDB(cfg)
	if err != nil {
		log.Fatalf("Failed to create vector database: %v", err)
	}

	if err := vectorDB.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize vector database: %v", err)
	}

	// Create and start the scheduler
	sched := scheduler.NewScheduler(sources, embeddingService, vectorDB)
	if err := sched.Start(ctx); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}

	// Create the assistant service
	assistant := service.NewAssistant(service.Config{
		OpenAIKey: os.Getenv("OPENAI_API_KEY"),
		ModelName: "gpt-4",
	}, embeddingService, vectorDB)

	// Set up HTTP handlers
	mux := http.NewServeMux()

	// Serve static files
	webDir := filepath.Join(".", "web")
	fs := http.FileServer(http.Dir(webDir))
	mux.Handle("/", fs)

	// API endpoint
	mux.HandleFunc("/ask", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		query := r.FormValue("query")
		if query == "" {
			http.Error(w, "Query parameter is required", http.StatusBadRequest)
			return
		}

		response, err := assistant.Ask(r.Context(), query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(response))
	})

	// Add CORS middleware
	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:8080"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type"},
	})

	// Start HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: corsHandler.Handler(mux),
	}

	go func() {
		log.Printf("Starting server on :8080")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down gracefully...")

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// Stop the scheduler
	sched.Stop()
}

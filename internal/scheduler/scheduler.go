package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/michaelgalloway/sophia/internal/database"
	"github.com/michaelgalloway/sophia/internal/datasources"
	"github.com/michaelgalloway/sophia/internal/embeddings"
	"github.com/robfig/cron/v3"
)

// Scheduler manages periodic data fetching from all sources
type Scheduler struct {
	cron             *cron.Cron
	sources          map[string]datasources.DataSource
	embeddingService embeddings.EmbeddingService
	vectorDB         database.VectorDB
	lastSync         map[string]time.Time
	mu               sync.RWMutex
}

// NewScheduler creates a new scheduler instance
func NewScheduler(
	sources map[string]datasources.DataSource,
	embeddingService embeddings.EmbeddingService,
	vectorDB database.VectorDB,
) *Scheduler {
	return &Scheduler{
		cron:             cron.New(),
		sources:          sources,
		embeddingService: embeddingService,
		vectorDB:         vectorDB,
		lastSync:         make(map[string]time.Time),
	}
}

// Start begins the scheduling of data fetching jobs
func (s *Scheduler) Start(ctx context.Context) error {

	s.vectorDB.DeleteAll(ctx)

	// Schedule hourly jobs for each source
	for name, source := range s.sources {
		source := source // Create new variable for closure
		name := name

		s.fetchAndProcess(ctx, source, name)

		_, err := s.cron.AddFunc("@hourly", func() {
			s.fetchAndProcess(ctx, source, name)
		})
		if err != nil {
			return err
		}
	}

	s.cron.Start()
	return nil
}

// Stop halts all scheduled jobs
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) fetchAndProcess(ctx context.Context, source datasources.DataSource, name string) {
	s.mu.RLock()
	since := s.lastSync[name]
	s.mu.RUnlock()

	log.Printf("Fetching %v", source.Name())

	docs, err := source.FetchData(ctx, since)
	if err != nil {
		// TODO: Add proper error handling/logging
		return
	}

	log.Printf("Found %d number of docs", len(docs))
	if len(docs) == 0 {
		return
	}

	log.Printf("Creating embeddings for %v", source.Name())
	vectors, err := s.embeddingService.CreateEmbeddings(ctx, docs)
	if err != nil {
		log.Printf("Error creating embedding: %v", err)
		return
	}
	log.Printf("Created %d embeddings for %v", len(vectors), source.Name())

	log.Printf("Storing embeddings for %v", source.Name())
	err = s.vectorDB.Store(ctx, docs, vectors)
	if err != nil {
		log.Printf("Error storing embedding: %v", err)
		return
	}
	log.Printf("Stored %d embeddings for %v", len(vectors), source.Name())

	s.mu.Lock()
	s.lastSync[name] = time.Now()
	s.mu.Unlock()
}

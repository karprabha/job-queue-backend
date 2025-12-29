package store

import (
	"context"
	"log"
	"time"
)

type Sweeper interface {
	Run(ctx context.Context)
}

type InMemorySweeper struct {
	jobStore JobStore
	interval time.Duration
	jobQueue chan string
}

func NewInMemorySweeper(jobStore JobStore, interval time.Duration, jobQueue chan string) *InMemorySweeper {
	return &InMemorySweeper{
		jobStore: jobStore,
		interval: interval,
		jobQueue: jobQueue,
	}
}

func (s *InMemorySweeper) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Sweeper: context canceled, shutting down.")
			return
		case <-ticker.C:
			if err := s.jobStore.RetryFailedJobs(ctx); err != nil {
				log.Printf("Sweeper: error retrying failed jobs: %v, retrying in %s", err, s.interval)
				continue
			}

			jobs, err := s.jobStore.GetPendingJobs(ctx)
			if err != nil {
				log.Printf("Sweeper: error getting pending jobs: %v, retrying in %s", err, s.interval)
				continue
			}

			log.Printf("Sweeper: fetched %d pending jobs", len(jobs))

			for _, job := range jobs {
				select {
				case <-ctx.Done():
					log.Println("Sweeper: context canceled, shutting down.")
					return
				case s.jobQueue <- job.ID:
					log.Printf("Sweeper: job %s added to queue", job.ID)
				default:
					log.Printf("Sweeper: job queue is full, job %s not added", job.ID)
				}
			}
		}
	}
}

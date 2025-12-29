package worker

import (
	"context"
	"log"
	"time"

	"github.com/karprabha/job-queue-backend/internal/domain"
	"github.com/karprabha/job-queue-backend/internal/store"
)

type Worker struct {
	id       int
	jobStore store.JobStore
	jobQueue chan string
}

func NewWorker(id int, jobStore store.JobStore, jobQueue chan string) *Worker {
	return &Worker{
		id:       id,
		jobStore: jobStore,
		jobQueue: jobQueue,
	}
}

func (w *Worker) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d shutting down", w.id)
			return
		case jobID, ok := <-w.jobQueue:
			if !ok {
				log.Printf("Worker %d shutting down because job queue is closed", w.id)
				return
			}
			job, err := w.jobStore.ClaimJob(ctx, jobID)

			if err != nil {
				log.Printf("Worker %d error claiming job: %s: %v", w.id, jobID, err)
				continue
			}

			if job == nil {
				log.Printf("Worker %d job %s already claimed or invalid", w.id, jobID)
				continue
			}

			log.Printf("Worker %d processing job %s", w.id, jobID)
			w.processJob(ctx, job)
		}
	}
}

func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		// Processing complete
	case <-ctx.Done():
		// Shutdown requested, abort processing
		log.Printf("Worker %d job %s processing aborted due to shutdown", w.id, job.ID)
		return
	}

	// Simulate failure deterministically
	if job.Type == "email" {
		// Signal failure to store
		shouldRetry, err := w.jobStore.MarkJobFailed(ctx, job.ID, "simulated failure for email job type")
		if err != nil {
			log.Printf("Worker %d error marking job failed: %s: %v", w.id, job.ID, err)
			return
		}

		if shouldRetry {
			// Store has already set it back to pending, re-enqueue
			select {
			case w.jobQueue <- job.ID:
				log.Printf("Worker %d re-queued job %s for retry (attempt %d/%d)", w.id, job.ID, job.Attempts+1, job.MaxRetries)
			case <-ctx.Done():
				log.Printf("Worker %d context cancelled while re-queuing job %s", w.id, job.ID)
			default:
				log.Printf("Worker %d job queue full, job %s will be picked up later", w.id, job.ID)
			}
		} else {
			log.Printf("Worker %d job %s permanently failed after %d attempts", w.id, job.ID, job.Attempts)
		}
		return
	}

	// Success - mark as completed
	err := w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusCompleted)
	if err != nil {
		log.Printf("Worker %d error updating job to completed: %s: %v", w.id, job.ID, err)
		return
	}
	log.Printf("Worker %d job %s completed successfully", w.id, job.ID)
}

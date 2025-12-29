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
		lastError := "Email sending failed"
		err := w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &lastError)
		if err != nil {
			log.Printf("Worker %d error updating job to completed: %s: %v", w.id, job.ID, err)
			return
		}
		log.Printf("Worker %d job %s failed", w.id, job.ID)
		return
	}

	// Success - mark as completed
	err := w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusCompleted, nil)
	if err != nil {
		log.Printf("Worker %d error updating job to completed: %s: %v", w.id, job.ID, err)
		return
	}
	log.Printf("Worker %d job %s completed successfully", w.id, job.ID)
}

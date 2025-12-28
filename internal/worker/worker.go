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

func (w *Worker) updateJobStatus(ctx context.Context, jobID string, status domain.JobStatus) {
	err := w.jobStore.UpdateStatus(ctx, jobID, status)
	if err != nil {
		log.Printf("Worker %d error updating job: %s: %v", w.id, jobID, err)
		return
	}
	log.Printf("Worker %d job %s updated to %s", w.id, jobID, status)
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

	w.updateJobStatus(ctx, job.ID, domain.StatusCompleted)
}

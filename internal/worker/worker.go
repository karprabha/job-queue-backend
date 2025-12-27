package worker

import (
	"context"
	"log"
	"time"

	"github.com/karprabha/job-queue-backend/internal/domain"
	"github.com/karprabha/job-queue-backend/internal/store"
)

type Worker struct {
	jobStore store.JobStore
	jobQueue chan *domain.Job
}

func NewWorker(jobStore store.JobStore, jobQueue chan *domain.Job) *Worker {
	return &Worker{
		jobStore: jobStore,
		jobQueue: jobQueue,
	}
}

func (w *Worker) Start(ctx context.Context, id int) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d shutting down", id)
			return
		case job, ok := <-w.jobQueue:
			if !ok {
				log.Printf("Worker %d shutting down because job queue is closed", id)
				return
			}
			claimed, err := w.jobStore.ClaimJob(ctx, job.ID)
			if err != nil {
				log.Printf("Worker %d error claiming job: %s: %v", id, job.ID, err)
				continue
			}

			if !claimed {
				log.Printf("Worker %d job %s not claimed", id, job.ID)
				continue
			}

			log.Printf("Worker %d processing job %s", id, job.ID)
			w.processJob(ctx, job, id)
		}
	}
}

func (w *Worker) updateJobStatus(ctx context.Context, job *domain.Job, status domain.JobStatus, id int) {
	job.Status = status
	err := w.jobStore.UpdateJob(ctx, job)
	if err != nil {
		log.Printf("Worker %d error updating job: %s: %v", id, job.ID, err)
		return
	}
	log.Printf("Worker %d job %s completed", id, job.ID)
}

func (w *Worker) processJob(ctx context.Context, job *domain.Job, id int) {
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		// Processing complete
	case <-ctx.Done():
		// Shutdown requested, abort processing
		log.Printf("Worker %d job %s processing aborted due to shutdown", id, job.ID)
		return
	}

	w.updateJobStatus(ctx, job, domain.StatusCompleted, id)
}

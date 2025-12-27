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
	jobQueue chan *domain.Job
}

func NewWorker(id int, jobStore store.JobStore, jobQueue chan *domain.Job) *Worker {
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
		case job, ok := <-w.jobQueue:
			if !ok {
				log.Printf("Worker %d shutting down because job queue is closed", w.id)
				return
			}
			claimed, err := w.jobStore.ClaimJob(ctx, job.ID)
			if err != nil {
				log.Printf("Worker %d error claiming job: %s: %v", w.id, job.ID, err)
				continue
			}

			if !claimed {
				log.Printf("Worker %d job %s not claimed", w.id, job.ID)
				continue
			}

			log.Printf("Worker %d processing job %s", w.id, job.ID)
			w.processJob(ctx, job)
		}
	}
}

func (w *Worker) updateJobStatus(ctx context.Context, job *domain.Job, status domain.JobStatus) {
	job.Status = status
	err := w.jobStore.UpdateJob(ctx, job)
	if err != nil {
		log.Printf("Worker %d error updating job: %s: %v", w.id, job.ID, err)
		return
	}
	log.Printf("Worker %d job %s updated to %s", w.id, job.ID, status)
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

	w.updateJobStatus(ctx, job, domain.StatusCompleted)
}

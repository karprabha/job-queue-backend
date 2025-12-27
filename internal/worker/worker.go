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

func (w *Worker) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-w.jobQueue:
			if !ok {
				return
			}
			claimed, err := w.jobStore.ClaimJob(ctx, job.ID)
			if err != nil {
				log.Printf("Error claiming job: %s: %v", job.ID, err)
				continue
			}

			if !claimed {
				continue
			}

			w.processJob(ctx, job)
		}
	}
}

func (w *Worker) updateJobStatus(ctx context.Context, job *domain.Job, status domain.JobStatus) {
	job.Status = status
	err := w.jobStore.UpdateJob(ctx, job)
	if err != nil {
		log.Printf("Error updating job: %s: %v", job.ID, err)
		return
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
		log.Printf("Job %s processing aborted due to shutdown", job.ID)
		w.updateJobStatus(ctx, job, domain.StatusFailed)
		return
	}

	w.updateJobStatus(ctx, job, domain.StatusCompleted)
}

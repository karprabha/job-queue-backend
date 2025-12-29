package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/karprabha/job-queue-backend/internal/domain"
	"github.com/karprabha/job-queue-backend/internal/store"
)

type Worker struct {
	id          int
	jobStore    store.JobStore
	metricStore store.MetricStore
	logger      *slog.Logger
	jobQueue    chan string
}

func NewWorker(id int, jobStore store.JobStore, metricStore store.MetricStore, logger *slog.Logger, jobQueue chan string) *Worker {
	return &Worker{
		id:          id,
		jobStore:    jobStore,
		metricStore: metricStore,
		logger:      logger,
		jobQueue:    jobQueue,
	}
}

func (w *Worker) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Worker shutting down", "worker_id", w.id)
			return
		case jobID, ok := <-w.jobQueue:
			if !ok {
				w.logger.Info("Worker shutting down because job queue is closed", "worker_id", w.id)
				return
			}
			job, err := w.jobStore.ClaimJob(ctx, jobID)

			if err != nil {
				w.logger.Error("Worker error claiming job", "worker_id", w.id, "job_id", jobID, "error", err)
				continue
			}

			if job == nil {
				w.logger.Info("Worker job already claimed or invalid", "worker_id", w.id, "job_id", jobID)
				continue
			}

			w.processJob(ctx, job)
		}
	}
}

func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	err := w.metricStore.IncrementJobsInProgress(ctx)
	if err != nil {
		w.logger.Error("Worker error incrementing jobs in progress", "worker_id", w.id, "error", err)
		return
	}
	w.logger.Info("Worker processing job", "worker_id", w.id, "job_id", job.ID)

	select {
	case <-timer.C:
		// Processing complete
	case <-ctx.Done():
		// Shutdown requested, abort processing
		w.logger.Info("Worker job processing aborted due to shutdown", "worker_id", w.id, "job_id", job.ID)
		return
	}

	// Simulate failure deterministically
	if job.Type == "email" {
		lastError := "Email sending failed"
		err := w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &lastError)
		if err != nil {
			w.logger.Error("Worker error updating job to failed", "worker_id", w.id, "job_id", job.ID, "error", err)
			return
		}
		w.logger.Info("Worker job failed", "worker_id", w.id, "job_id", job.ID)

		err = w.metricStore.IncrementJobsFailed(ctx)
		if err != nil {
			w.logger.Error("Worker error incrementing jobs failed", "worker_id", w.id, "error", err)
			return
		}

		return
	}

	// Success - mark as completed
	err = w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusCompleted, nil)
	if err != nil {
		w.logger.Error("Worker error updating job to completed", "worker_id", w.id, "job_id", job.ID, "error", err)
		return
	}
	err = w.metricStore.IncrementJobsCompleted(ctx)
	if err != nil {
		w.logger.Error("Worker error incrementing jobs completed", "worker_id", w.id, "error", err)
		return
	}
	w.logger.Info("Worker job completed successfully", "worker_id", w.id, "job_id", job.ID)
}

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
	w.logger.Info("Worker started", "event", "worker_started", "worker_id", w.id)
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Worker shutting down", "event", "worker_stopped", "worker_id", w.id)
			return
		case jobID, ok := <-w.jobQueue:
			if !ok {
				w.logger.Info("Worker shutting down because job queue is closed", "event", "worker_stopped", "worker_id", w.id)
				return
			}
			job, err := w.jobStore.ClaimJob(ctx, jobID)

			if err != nil {
				w.logger.Error("Worker error claiming job", "event", "job_claim_error", "worker_id", w.id, "job_id", jobID, "error", err)
				continue
			}

			if job == nil {
				w.logger.Info("Worker job already claimed or invalid", "event", "job_claim_failed", "worker_id", w.id, "job_id", jobID)
				continue
			}

			w.logger.Info("Job started", "event", "job_started", "worker_id", w.id, "job_id", jobID)
			w.processJob(ctx, job)
		}
	}
}

func (w *Worker) processJob(ctx context.Context, job *domain.Job) {
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	err := w.metricStore.IncrementJobsInProgress(ctx)
	if err != nil {
		w.logger.Error("Worker error incrementing jobs in progress", "event", "metric_error", "worker_id", w.id, "error", err)
		return
	}

	select {
	case <-timer.C:
		// Processing complete
	case <-ctx.Done():
		// Shutdown requested, abort processing
		w.logger.Info("Worker job processing aborted due to shutdown", "event", "job_aborted", "worker_id", w.id, "job_id", job.ID)
		return
	}

	// Simulate failure deterministically
	if job.Type == "email" {
		lastError := "Email sending failed"
		err := w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusFailed, &lastError)
		if err != nil {
			w.logger.Error("Worker error updating job to failed", "event", "job_update_error", "worker_id", w.id, "job_id", job.ID, "error", err)
			return
		}
		w.logger.Info("Job failed", "event", "job_failed", "worker_id", w.id, "job_id", job.ID)

		err = w.metricStore.IncrementJobsFailed(ctx)
		if err != nil {
			w.logger.Error("Worker error incrementing jobs failed", "event", "metric_error", "worker_id", w.id, "error", err)
			return
		}

		return
	}

	// Success - mark as completed
	err = w.jobStore.UpdateStatus(ctx, job.ID, domain.StatusCompleted, nil)
	if err != nil {
		w.logger.Error("Worker error updating job to completed", "event", "job_update_error", "worker_id", w.id, "job_id", job.ID, "error", err)
		return
	}
	err = w.metricStore.IncrementJobsCompleted(ctx)
	if err != nil {
		w.logger.Error("Worker error incrementing jobs completed", "event", "metric_error", "worker_id", w.id, "error", err)
		return
	}
	w.logger.Info("Job completed", "event", "job_completed", "worker_id", w.id, "job_id", job.ID)
}

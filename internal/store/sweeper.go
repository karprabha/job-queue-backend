package store

import (
	"context"
	"log/slog"
	"time"
)

type Sweeper interface {
	Run(ctx context.Context)
}

type InMemorySweeper struct {
	jobStore    JobStore
	metricStore MetricStore
	logger      *slog.Logger
	interval    time.Duration
	jobQueue    chan string
}

func NewInMemorySweeper(jobStore JobStore, metricStore MetricStore, logger *slog.Logger, interval time.Duration, jobQueue chan string) *InMemorySweeper {
	return &InMemorySweeper{
		jobStore:    jobStore,
		metricStore: metricStore,
		logger:      logger,
		interval:    interval,
		jobQueue:    jobQueue,
	}
}

func (s *InMemorySweeper) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Sweeper shutting down", "event", "sweeper_stopped")
			return
		case <-ticker.C:
			if err := s.jobStore.RetryFailedJobs(ctx, s.metricStore, s.logger); err != nil {
				s.logger.Error("Sweeper error retrying failed jobs", "event", "sweeper_error", "error", err)
				continue
			}

			jobs, err := s.jobStore.GetPendingJobs(ctx)
			if err != nil {
				s.logger.Error("Sweeper error getting pending jobs", "event", "sweeper_error", "error", err)
				continue
			}

			for _, job := range jobs {
				select {
				case <-ctx.Done():
					s.logger.Info("Sweeper shutting down", "event", "sweeper_stopped")
					return
				case s.jobQueue <- job.ID:
					s.logger.Info("Job enqueued by sweeper", "event", "job_enqueued", "job_id", job.ID)
				default:
					s.logger.Info("Job queue is full, job not added", "event", "job_enqueue_failed", "job_id", job.ID)
				}
			}
		}
	}
}

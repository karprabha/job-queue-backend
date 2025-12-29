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
			s.logger.Info("Sweeper: context canceled, shutting down.")
			return
		case <-ticker.C:
			if err := s.jobStore.RetryFailedJobs(ctx, s.metricStore, s.logger); err != nil {
				s.logger.Error("Sweeper: error retrying failed jobs", "error", err, "retrying in", s.interval)
				continue
			}

			jobs, err := s.jobStore.GetPendingJobs(ctx)
			if err != nil {
				s.logger.Error("Sweeper: error getting pending jobs", "error", err, "retrying in", s.interval)
				continue
			}

			s.logger.Info("Sweeper: fetched pending jobs", "count", len(jobs))

			for _, job := range jobs {
				select {
				case <-ctx.Done():
					s.logger.Info("Sweeper: context canceled, shutting down.")
					return
				case s.jobQueue <- job.ID:
					s.logger.Info("Sweeper: job added to queue", "job_id", job.ID)
				default:
					s.logger.Info("Sweeper: job queue is full, job not added", "job_id", job.ID)
				}
			}
		}
	}
}

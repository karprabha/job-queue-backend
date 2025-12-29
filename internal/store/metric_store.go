package store

import (
	"context"
	"sync"

	"github.com/karprabha/job-queue-backend/internal/domain"
)

type MetricStore interface {
	GetMetrics(ctx context.Context) (*domain.Metric, error)
	IncrementJobsCreated(ctx context.Context) error
	IncrementJobsCompleted(ctx context.Context) error
	IncrementJobsFailed(ctx context.Context) error
	IncrementJobsRetried(ctx context.Context) error
	IncrementJobsInProgress(ctx context.Context) error
}

type InMemoryMetricStore struct {
	mu      sync.RWMutex
	metrics *domain.Metric
}

func NewInMemoryMetricStore() *InMemoryMetricStore {
	return &InMemoryMetricStore{
		metrics: domain.NewMetric(),
	}
}

func (s *InMemoryMetricStore) GetMetrics(ctx context.Context) (*domain.Metric, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		s.mu.RLock()
		defer s.mu.RUnlock()
		return s.metrics, nil
	}
}

func (s *InMemoryMetricStore) IncrementJobsCreated(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		s.metrics.TotalJobsCreated++
		return nil
	}

}

func (s *InMemoryMetricStore) IncrementJobsCompleted(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		s.metrics.JobsCompleted++
		s.metrics.JobsInProgress--
		return nil
	}
}

func (s *InMemoryMetricStore) IncrementJobsFailed(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		s.metrics.JobsFailed++
		s.metrics.JobsInProgress--
		return nil
	}
}

func (s *InMemoryMetricStore) IncrementJobsRetried(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		s.metrics.JobsRetried++
		s.metrics.JobsFailed--
		return nil
	}
}

func (s *InMemoryMetricStore) IncrementJobsInProgress(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		s.mu.Lock()
		defer s.mu.Unlock()

		s.metrics.JobsInProgress++
		return nil
	}
}

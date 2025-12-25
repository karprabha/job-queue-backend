package store

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/karprabha/job-queue-backend/internal/domain"
)

type JobStore interface {
	CreateJob(ctx context.Context, job domain.Job) error
	DeleteJob(ctx context.Context, id string) error
	GetJobs(ctx context.Context) ([]domain.Job, error)
	GetJob(ctx context.Context, id string) (domain.Job, error)
}

type InMemoryJobStore struct {
	jobs map[string]domain.Job
	mu   sync.RWMutex
}

func NewInMemoryJobStore() *InMemoryJobStore {
	return &InMemoryJobStore{
		jobs: make(map[string]domain.Job),
		mu:   sync.RWMutex{},
	}
}

func (s *InMemoryJobStore) CreateJob(ctx context.Context, job domain.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.jobs[job.ID] = job

	return nil
}

func (s *InMemoryJobStore) DeleteJob(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	delete(s.jobs, id)

	return nil
}

func (s *InMemoryJobStore) GetJobs(ctx context.Context) ([]domain.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	jobs := make([]domain.Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
	})

	return jobs, nil
}

func (s *InMemoryJobStore) GetJob(ctx context.Context, id string) (domain.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	select {
	case <-ctx.Done():
		return domain.Job{}, ctx.Err()
	default:
	}

	job, ok := s.jobs[id]
	if !ok {
		return domain.Job{}, fmt.Errorf("job with id %s not found", id)
	}
	return job, nil
}

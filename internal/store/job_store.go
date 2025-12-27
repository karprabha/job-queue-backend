package store

import (
	"context"
	"sort"
	"sync"

	"github.com/karprabha/job-queue-backend/internal/domain"
)

type JobStore interface {
	CreateJob(ctx context.Context, job *domain.Job) error
	GetJobs(ctx context.Context) ([]domain.Job, error)
	UpdateJob(ctx context.Context, job *domain.Job) error
	ClaimJob(ctx context.Context, jobID string) (bool, error)
}

type InMemoryJobStore struct {
	jobs map[string]domain.Job
	mu   sync.RWMutex
}

func NewInMemoryJobStore() *InMemoryJobStore {
	return &InMemoryJobStore{
		jobs: make(map[string]domain.Job),
	}
}

func (s *InMemoryJobStore) CreateJob(ctx context.Context, job *domain.Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs[job.ID] = *job

	return nil
}

func (s *InMemoryJobStore) GetJobs(ctx context.Context) ([]domain.Job, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]domain.Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}

	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
	})

	return jobs, nil
}

func (s *InMemoryJobStore) UpdateJob(ctx context.Context, job *domain.Job) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs[job.ID] = *job

	return nil
}

func (s *InMemoryJobStore) ClaimJob(ctx context.Context, jobID string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok || job.Status != domain.StatusPending {
		return false, nil
	}

	job.Status = domain.StatusProcessing
	s.jobs[jobID] = job

	return true, nil
}

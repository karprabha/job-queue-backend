package store

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/karprabha/job-queue-backend/internal/domain"
)

type JobStore interface {
	CreateJob(ctx context.Context, job *domain.Job) error
	GetJobs(ctx context.Context) ([]domain.Job, error)
	ClaimJob(ctx context.Context, jobID string) (*domain.Job, error)
	UpdateStatus(ctx context.Context, jobID string, status domain.JobStatus, lastError *string) error
	GetFailedJobs(ctx context.Context) ([]domain.Job, error)
	GetPendingJobs(ctx context.Context) ([]domain.Job, error)
	RetryFailedJobs(ctx context.Context) error
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

func canTransition(from, to domain.JobStatus) bool {
	switch {
	case from == domain.StatusPending && to == domain.StatusProcessing:
		return true
	case from == domain.StatusProcessing && to == domain.StatusCompleted:
		return true
	case from == domain.StatusProcessing && to == domain.StatusFailed:
		return true
	case from == domain.StatusFailed && to == domain.StatusPending:
		return true
	default:
		return false
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

func (s *InMemoryJobStore) ClaimJob(ctx context.Context, jobID string) (*domain.Job, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok || job.Status != domain.StatusPending {
		return nil, nil
	}

	job.Status = domain.StatusProcessing
	job.Attempts++
	s.jobs[jobID] = job

	jobCopy := job

	return &jobCopy, nil
}

func (s *InMemoryJobStore) UpdateStatus(ctx context.Context, jobID string, status domain.JobStatus, lastError *string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return errors.New("job not found in store")
	}

	// Validate transition
	if !canTransition(job.Status, status) {
		return errors.New("invalid state transition")
	}

	job.Status = status
	if lastError != nil {
		job.LastError = lastError
	}
	s.jobs[jobID] = job

	return nil
}

func (s *InMemoryJobStore) GetFailedJobs(ctx context.Context) ([]domain.Job, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]domain.Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		if job.Status == domain.StatusFailed {
			jobs = append(jobs, job)
		}
	}

	return jobs, nil
}

func (s *InMemoryJobStore) GetPendingJobs(ctx context.Context) ([]domain.Job, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]domain.Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		if job.Status == domain.StatusPending {
			jobs = append(jobs, job)
		}
	}

	return jobs, nil
}

func (s *InMemoryJobStore) RetryFailedJobs(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for jobID, job := range s.jobs {
		if job.Status == domain.StatusFailed && job.Attempts <= job.MaxRetries {
			job.Status = domain.StatusPending
			s.jobs[jobID] = job
		}
	}

	return nil
}

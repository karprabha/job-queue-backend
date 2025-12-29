package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
)

type Job struct {
	ID         string
	Type       string
	Status     JobStatus
	Payload    json.RawMessage
	MaxRetries int
	Attempts   int
	LastError  *string
	CreatedAt  time.Time
}

func NewJob(jobType string, jobPayload json.RawMessage) *Job {
	const attempts = 0
	const maxRetries = 3

	job := &Job{
		ID:         uuid.New().String(),
		Type:       jobType,
		Status:     StatusPending,
		Payload:    jobPayload,
		MaxRetries: maxRetries,
		Attempts:   attempts,
		LastError:  nil,
		CreatedAt:  time.Now().UTC(),
	}

	return job
}

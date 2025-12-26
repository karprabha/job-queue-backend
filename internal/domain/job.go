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
)

type Job struct {
	ID        string
	Type      string
	Status    JobStatus
	Payload   json.RawMessage
	CreatedAt time.Time
}

func NewJob(jobType string, jobPayload json.RawMessage) *Job {
	job := &Job{
		ID:        uuid.New().String(),
		Type:      jobType,
		Status:    StatusPending,
		Payload:   jobPayload,
		CreatedAt: time.Now().UTC(),
	}

	return job
}

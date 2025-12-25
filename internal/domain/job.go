package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Job struct {
	ID        string
	Type      string
	Status    string
	Payload   json.RawMessage
	CreatedAt time.Time
}

const (
	StatusPending = "pending"
)

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

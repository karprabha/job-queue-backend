package domain

import (
	"time"

	"github.com/google/uuid"
)

type Payload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
}

type Job struct {
	ID        string
	Type      string
	Status    string
	CreatedAt time.Time
}

func NewJob(jobType string, jobPayload Payload) (*Job, error) {
	job := &Job{
		ID:        uuid.New().String(),
		Type:      jobType,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	return job, nil
}

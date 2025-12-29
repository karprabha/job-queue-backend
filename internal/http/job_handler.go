package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/karprabha/job-queue-backend/internal/domain"
	"github.com/karprabha/job-queue-backend/internal/store"
)

type JobHandler struct {
	store       store.JobStore
	metricStore store.MetricStore
	logger      *slog.Logger
	jobQueue    chan string
}

func NewJobHandler(store store.JobStore, metricStore store.MetricStore, logger *slog.Logger, jobQueue chan string) *JobHandler {
	return &JobHandler{
		store:       store,
		metricStore: metricStore,
		logger:      logger,
		jobQueue:    jobQueue,
	}
}

type CreateJobRequest struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
type JobResponse struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func jobToResponse(job *domain.Job) JobResponse {
	return JobResponse{
		ID:        job.ID,
		Type:      job.Type,
		Status:    string(job.Status),
		CreatedAt: job.CreatedAt.Format(time.RFC3339),
	}
}

func (h *JobHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024) // 1MB max

	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		// Detect if it's too large
		if strings.Contains(err.Error(), "request body too large") {
			ErrorResponse(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}

		ErrorResponse(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	var request CreateJobRequest
	if err := json.Unmarshal(bodyBytes, &request); err != nil {
		ErrorResponse(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	if request.Type == "" {
		ErrorResponse(w, "Job type is required and must be non-empty", http.StatusBadRequest)
		return
	}

	job := domain.NewJob(request.Type, request.Payload)

	err = h.store.CreateJob(r.Context(), job)
	if err != nil {
		ErrorResponse(w, "Failed to create job", http.StatusInternalServerError)
		return
	}
	h.logger.Info("Job created", "job_id", job.ID)

	err = h.metricStore.IncrementJobsCreated(r.Context())
	if err != nil {
		h.logger.Error("Failed to increment jobs created", "error", err)
	}

	select {
	case h.jobQueue <- job.ID:
		h.logger.Info("Job added to queue", "job_id", job.ID)
	case <-r.Context().Done():
		ErrorResponse(w, "Request cancelled", http.StatusRequestTimeout)
		return
	default:
		h.logger.Error("Failed to add job to queue", "error", err)
	}

	response := jobToResponse(job)

	responseBytes, err := json.Marshal(response)
	if err != nil {
		ErrorResponse(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if _, err := w.Write(responseBytes); err != nil {
		return
	}
}

func (h *JobHandler) GetJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := h.store.GetJobs(r.Context())
	if err != nil {
		ErrorResponse(w, "Failed to get jobs", http.StatusInternalServerError)
		return
	}

	response := make([]JobResponse, len(jobs))
	for i, job := range jobs {
		response[i] = jobToResponse(&job)
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		ErrorResponse(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(responseBytes); err != nil {
		return
	}
}

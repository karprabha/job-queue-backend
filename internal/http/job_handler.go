package http

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/karprabha/job-queue-backend/internal/domain"
)

type CreateJobRequest struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
type CreateJobResponse struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
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

	response := CreateJobResponse{
		ID:        job.ID,
		Type:      job.Type,
		Status:    string(job.Status),
		CreatedAt: job.CreatedAt.Format(time.RFC3339),
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		ErrorResponse(w, "Marshal failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if _, err := w.Write(responseBytes); err != nil {
		return
	}
}

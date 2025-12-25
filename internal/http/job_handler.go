package http

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/karprabha/job-queue-backend/internal/domain"
	"github.com/karprabha/job-queue-backend/internal/utils"
)

type CreateJobRequest struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"` // Accept any JSON
}
type CreateJobResponse struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		utils.ErrorResponse(w, "Read failed", http.StatusInternalServerError)
		return
	}

	var request CreateJobRequest
	if err := json.Unmarshal(bodyBytes, &request); err != nil {
		utils.ErrorResponse(w, "Parse failed", http.StatusBadRequest)
		return
	}

	if request.Type == "" {
		utils.ErrorResponse(w, "Job type is required", http.StatusBadRequest)
		return
	}

	job := domain.NewJob(request.Type, request.Payload)

	response := CreateJobResponse{
		ID:        job.ID,
		Type:      job.Type,
		Status:    job.Status,
		CreatedAt: job.CreatedAt.Format(time.RFC3339),
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		utils.ErrorResponse(w, "Marshal failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(responseBytes); err != nil {
		utils.ErrorResponse(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

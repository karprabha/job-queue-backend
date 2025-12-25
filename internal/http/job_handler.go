package http

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/karprabha/job-queue-backend/internal/domain"
)

type CreateJobRequest struct {
	Type    string         `json:"type"`
	Payload domain.Payload `json:"payload"`
}

type CreateJobResponse struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

func isValidJSON(payload string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(payload), &js) == nil
}

func ErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	jsonBytes, err := json.Marshal(map[string]string{"error": message})

	if err != nil {
		http.Error(w, "Failed to marshal error response", statusCode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if _, err := w.Write(jsonBytes); err != nil {
		http.Error(w, "Failed to write error response", statusCode)
		return
	}
}

func CreateJobHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	select {
	case <-ctx.Done():
		ErrorResponse(w, "Context cancelled", http.StatusInternalServerError)
		return
	default:
		// continue with the request
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		ErrorResponse(w, "Read failed", 500)
		return
	}

	var request CreateJobRequest
	if err := json.Unmarshal(bodyBytes, &request); err != nil {
		ErrorResponse(w, "Parse failed", 400)
		return
	}

	if request.Type == "" {
		ErrorResponse(w, "Job type is required", 400)
		return
	}

	payloadBytes, err := json.Marshal(request.Payload)
	if err != nil || !isValidJSON(string(payloadBytes)) {
		ErrorResponse(w, "Job payload is not valid JSON", 400)
		return
	}

	if request.Payload.To == "" || request.Payload.Subject == "" {
		ErrorResponse(w, "Job payload is required and must contain to and subject", 400)
		return
	}

	job, err := domain.NewJob(request.Type, request.Payload)
	if err != nil {
		ErrorResponse(w, "New job failed", 500)
		return
	}

	responseBytes, err := json.Marshal(job)
	if err != nil {
		ErrorResponse(w, "Marshal failed", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if _, err := w.Write(responseBytes); err != nil {
		ErrorResponse(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

package http

import (
	"encoding/json"
	"net/http"

	"github.com/karprabha/job-queue-backend/internal/utils"
)

type HealthCheckResponse struct {
	Status string `json:"status"`
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	responseData := HealthCheckResponse{
		Status: "ok",
	}

	jsonBytes, err := json.Marshal(responseData)
	if err != nil {
		utils.ErrorResponse(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write(jsonBytes); err != nil {
		utils.ErrorResponse(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}

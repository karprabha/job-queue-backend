package http

import (
	"encoding/json"
	"net/http"
)

type HealthCheckResponse struct {
	Status string `json:"status"`
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	responseData := HealthCheckResponse{
		Status: "ok",
	}

	// Set headers BEFORE encoding
	w.Header().Set("Content-Type", "application/json")

	// Encode directly - if it fails, http.Error can still set status
	if err := json.NewEncoder(w).Encode(responseData); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	// Status 200 is implicit if no error
}

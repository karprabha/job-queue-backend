package http

import (
	"encoding/json"
	"net/http"
)

type HealthCheckResponse struct {
	Status string `json:"status"`
}

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()

	select {
	case <-ctx.Done():
		http.Error(w, "Context cancelled", http.StatusInternalServerError)
		return
	default:
		// continue with the request
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	responseData := HealthCheckResponse{
		Status: "ok",
	}

	jsonBytes, err := json.Marshal(responseData)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

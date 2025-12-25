package http

import (
	"encoding/json"
	"net/http"
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
		ErrorResponse(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(jsonBytes); err != nil {
		return
	}
}

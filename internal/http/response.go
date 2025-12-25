package http

import (
	"encoding/json"
	"net/http"
)

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

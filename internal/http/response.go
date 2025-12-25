package http

import (
	"encoding/json"
	"net/http"
)

func ErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	jsonBytes, err := json.Marshal(map[string]string{"error": message})
	if err != nil {
		// If we can't marshal, fall back to plain text error
		// Headers haven't been written yet, so http.Error is safe
		http.Error(w, "Failed to marshal error response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if _, err := w.Write(jsonBytes); err != nil {
		// Headers already written, can't send another response
		// Client may have disconnected - just return
		return
	}
}

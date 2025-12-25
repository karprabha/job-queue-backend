package main

import (
	"log"
	"net/http"

	internalhttp "github.com/karprabha/job-queue-backend/internal/http"
)

func main() {
	http.HandleFunc("/health", internalhttp.HealthCheckHandler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	internalhttp "github.com/karprabha/job-queue-backend/internal/http"
)

func main() {
	// 1. Read port from env
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 2. Set up routes
	http.HandleFunc("/health", internalhttp.HealthCheckHandler)

	// 3. Create http.Server instance
	srv := &http.Server{
		Addr: ":" + port,
	}

	// 4. Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// 5. Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 6. Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down...")

	// 7. Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

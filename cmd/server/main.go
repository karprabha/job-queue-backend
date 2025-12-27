package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/karprabha/job-queue-backend/internal/domain"
	internalhttp "github.com/karprabha/job-queue-backend/internal/http"
	"github.com/karprabha/job-queue-backend/internal/store"
	"github.com/karprabha/job-queue-backend/internal/worker"
)

type Config struct {
	Port             string
	JobQueueCapacity int
	WorkerCount      int
}

func NewConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	jobQueueCapacity := os.Getenv("JOB_QUEUE_CAPACITY")
	if jobQueueCapacity == "" {
		jobQueueCapacity = "100"
	}

	workerCount := os.Getenv("WORKER_COUNT")
	if workerCount == "" {
		workerCount = "10"
	}

	workerCountInt, err := strconv.Atoi(workerCount)
	if err != nil {
		workerCountInt = 10
	}

	jobQueueCapacityInt, err := strconv.Atoi(jobQueueCapacity)
	if err != nil {
		jobQueueCapacityInt = 100
	}

	return &Config{
		Port:             port,
		JobQueueCapacity: jobQueueCapacityInt,
		WorkerCount:      workerCountInt,
	}
}

func main() {
	// 1. Read port from env
	config := NewConfig()

	jobStore := store.NewInMemoryJobStore()

	jobQueue := make(chan *domain.Job, config.JobQueueCapacity)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	var wg sync.WaitGroup

	for i := 0; i < config.WorkerCount; i++ {
		worker := worker.NewWorker(jobStore, jobQueue)
		wg.Go(func() {
			worker.Start(workerCtx, i)
		})
	}

	mux := http.NewServeMux()

	jobHandler := internalhttp.NewJobHandler(jobStore, jobQueue)

	// Health Route
	mux.HandleFunc("GET /health", internalhttp.HealthCheckHandler)

	// Job Routes
	mux.HandleFunc("GET /jobs", jobHandler.GetJobs)
	mux.HandleFunc("POST /jobs", jobHandler.CreateJob)

	// Create http.Server instance
	srv := &http.Server{
		Addr:    ":" + config.Port,
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down...")

	// Cancel the context to stop the worker
	workerCancel()
	wg.Wait()
	close(jobQueue)

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

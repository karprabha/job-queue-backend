package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/karprabha/job-queue-backend/internal/config"
	internalhttp "github.com/karprabha/job-queue-backend/internal/http"
	"github.com/karprabha/job-queue-backend/internal/store"
	"github.com/karprabha/job-queue-backend/internal/worker"
)

func main() {
	config := config.NewConfig()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	jobStore := store.NewInMemoryJobStore()
	metricStore := store.NewInMemoryMetricStore()

	jobQueue := make(chan string, config.JobQueueCapacity)

	sweeper := store.NewInMemorySweeper(jobStore, metricStore, logger, config.SweeperInterval, jobQueue)

	sweeperCtx, sweeperCancel := context.WithCancel(context.Background())
	defer sweeperCancel()

	var sweeperWg sync.WaitGroup
	sweeperWg.Go(func() {
		sweeper.Run(sweeperCtx)
	})

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	var wg sync.WaitGroup

	for i := 0; i < config.WorkerCount; i++ {
		workerID := i // Capture loop variable to avoid closure issue
		worker := worker.NewWorker(workerID, jobStore, metricStore, logger, jobQueue)
		wg.Go(func() {
			worker.Start(workerCtx)
		})
	}

	mux := http.NewServeMux()

	metricHandler := internalhttp.NewMetricHandler(metricStore, logger)
	jobHandler := internalhttp.NewJobHandler(jobStore, metricStore, logger, jobQueue)

	// Health Route
	mux.HandleFunc("GET /health", internalhttp.HealthCheckHandler)

	// Job Routes
	mux.HandleFunc("GET /jobs", jobHandler.GetJobs)
	mux.HandleFunc("POST /jobs", jobHandler.CreateJob)

	// Metric Routes
	mux.HandleFunc("GET /metrics", metricHandler.GetMetrics)

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
	logger.Info("Shutting down...")

	// 1. Shutdown HTTP server first (stops accepting new requests, waits for in-flight)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown error", "error", err)
	}

	// 2. Cancel sweeper and wait
	sweeperCancel()
	sweeperWg.Wait()

	// 3. Close the job queue (no more requests can enqueue)
	close(jobQueue)

	// 4. Cancel workers and wait
	workerCancel()
	wg.Wait()

	logger.Info("Server stopped")
}

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
	"github.com/karprabha/job-queue-backend/internal/recovery"
	"github.com/karprabha/job-queue-backend/internal/store"
	"github.com/karprabha/job-queue-backend/internal/worker"
)

func main() {
	config := config.NewConfig()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// 1. Initialize store
	jobStore := store.NewInMemoryJobStore()
	metricStore := store.NewInMemoryMetricStore()

	// 2. Run recovery logic (BEFORE queue initialization and workers)
	// Initialize queue for recovery (but workers not started yet)
	jobQueue := make(chan string, config.JobQueueCapacity)

	recoveryCtx := context.Background()
	if err := recovery.RecoverJobs(recoveryCtx, jobStore, jobQueue, logger); err != nil {
		log.Fatalf("Recovery failed: %v", err)
	}

	// 3. Initialize queue (already done above)
	// Queue is ready, now we can start workers

	// 4. Start workers
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	// Create shutdown context for handlers to check shutdown state
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	defer shutdownCancel()

	var wg sync.WaitGroup

	for i := 0; i < config.WorkerCount; i++ {
		workerID := i // Capture loop variable to avoid closure issue
		worker := worker.NewWorker(workerID, jobStore, metricStore, logger, jobQueue)
		wg.Go(func() {
			worker.Start(workerCtx)
		})
	}

	// Start sweeper (runs periodically to retry failed jobs and enqueue pending)
	sweeper := store.NewInMemorySweeper(jobStore, metricStore, logger, config.SweeperInterval, jobQueue)

	sweeperCtx, sweeperCancel := context.WithCancel(context.Background())
	defer sweeperCancel()

	var sweeperWg sync.WaitGroup
	sweeperWg.Go(func() {
		sweeper.Run(sweeperCtx)
	})

	mux := http.NewServeMux()

	metricHandler := internalhttp.NewMetricHandler(metricStore, logger)
	jobHandler := internalhttp.NewJobHandler(jobStore, metricStore, logger, jobQueue, shutdownCtx)

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

	// 1. Signal shutdown to handlers (they will reject new jobs)
	shutdownCancel()
	logger.Info("Shutdown signal sent to handlers")

	// 2. Shutdown HTTP server (stops accepting new requests, waits for in-flight)
	serverShutdownCtx, serverShutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer serverShutdownCancel()

	if err := srv.Shutdown(serverShutdownCtx); err != nil {
		if err == context.DeadlineExceeded {
			logger.Warn("Server shutdown timeout exceeded, forcing close")
		} else {
			logger.Error("Server shutdown error", "error", err)
		}
	}

	// 3. Cancel sweeper and wait
	sweeperCancel()
	sweeperWg.Wait()
	logger.Info("Sweeper stopped")

	// 4. Cancel workers (stops picking new jobs) and wait for them to finish current jobs
	workerCancel()
	wg.Wait()
	logger.Info("Workers stopped")

	// 5. Close the job queue (safe now that workers are done)
	close(jobQueue)

	logger.Info("Server stopped")
}

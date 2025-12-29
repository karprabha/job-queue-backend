package recovery

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/karprabha/job-queue-backend/internal/domain"
	"github.com/karprabha/job-queue-backend/internal/store"
)

// RecoverJobs performs startup recovery:
// 1. Moves processing jobs back to pending (they were interrupted during crash)
// 2. Re-enqueues all pending jobs (including newly recovered ones)
// 3. Respects backpressure (waits if queue is full, no jobs dropped)
func RecoverJobs(
	ctx context.Context,
	jobStore store.JobStore,
	jobQueue chan string,
	logger *slog.Logger,
) error {
	logger.Info("Starting recovery", "event", "recovery_started")

	// Step 1: Move processing jobs back to pending
	// These jobs were in-flight when the process crashed
	processingJobs, err := jobStore.GetProcessingJobs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get processing jobs: %w", err)
	}

	processingRecovered := 0
	for _, job := range processingJobs {
		// Use UpdateStatus to respect state transition rules
		err := jobStore.UpdateStatus(ctx, job.ID, domain.StatusPending, nil)
		if err != nil {
			logger.Error("Failed to recover processing job",
				"event", "recovery_error",
				"job_id", job.ID,
				"error", err)
			// Continue with other jobs - don't fail entire recovery
			continue
		}
		processingRecovered++
		logger.Info("Recovered processing job",
			"event", "job_recovered",
			"job_id", job.ID)
	}

	// Step 2: Re-enqueue all pending jobs (including newly recovered ones)
	pendingJobs, err := jobStore.GetPendingJobs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pending jobs: %w", err)
	}

	pendingReEnqueued := 0
	for _, job := range pendingJobs {
		if err := reEnqueueWithBackpressure(ctx, job.ID, jobQueue, logger); err != nil {
			return fmt.Errorf("failed to re-enqueue job %s: %w", job.ID, err)
		}
		pendingReEnqueued++
	}

	logger.Info("Recovery completed",
		"event", "recovery_completed",
		"processing_recovered", processingRecovered,
		"pending_re_enqueued", pendingReEnqueued)

	return nil
}

// reEnqueueWithBackpressure attempts to enqueue a job with exponential backoff
// if the queue is full. This ensures no jobs are dropped during recovery.
func reEnqueueWithBackpressure(
	ctx context.Context,
	jobID string,
	jobQueue chan string,
	logger *slog.Logger,
) error {
	backoff := 50 * time.Millisecond
	maxBackoff := 5 * time.Second
	maxAttempts := 10

	for attempt := 0; attempt < maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case jobQueue <- jobID:
			if attempt > 0 {
				logger.Info("Job re-enqueued after backoff",
					"event", "job_re_enqueued",
					"job_id", jobID,
					"attempt", attempt+1)
			}
			return nil // Success!
		default:
			if attempt < maxAttempts-1 {
				logger.Info("Queue full during recovery, backing off",
					"event", "recovery_backpressure",
					"job_id", jobID,
					"attempt", attempt+1,
					"backoff_ms", backoff.Milliseconds())

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
					// Exponential backoff with cap
					backoff = time.Duration(float64(backoff) * 1.5)
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
				}
			}
		}
	}

	return fmt.Errorf("failed to enqueue job %s after %d attempts: queue persistently full", jobID, maxAttempts)
}

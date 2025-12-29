package http

import (
	"encoding/json"
	"net/http"

	"github.com/karprabha/job-queue-backend/internal/store"
)

type MetricHandler struct {
	metricStore store.MetricStore
}

func NewMetricHandler(metricStore store.MetricStore) *MetricHandler {
	return &MetricHandler{metricStore: metricStore}
}

type MetricResponse struct {
	TotalJobsCreated int `json:"total_jobs_created"`
	JobsCompleted    int `json:"jobs_completed"`
	JobsFailed       int `json:"jobs_failed"`
	JobsRetried      int `json:"jobs_retried"`
	JobsInProgress   int `json:"jobs_in_progress"`
}

func (h *MetricHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.metricStore.GetMetrics(r.Context())
	if err != nil {
		ErrorResponse(w, "Failed to get metrics", http.StatusInternalServerError)
		return
	}

	response := MetricResponse{
		TotalJobsCreated: metrics.TotalJobsCreated,
		JobsCompleted:    metrics.JobsCompleted,
		JobsFailed:       metrics.JobsFailed,
		JobsRetried:      metrics.JobsRetried,
		JobsInProgress:   metrics.JobsInProgress,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		ErrorResponse(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(responseBytes); err != nil {
		return
	}
}

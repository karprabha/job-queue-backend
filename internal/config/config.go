package config

import (
	"os"
	"strconv"
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

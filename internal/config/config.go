package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port             string
	JobQueueCapacity int
	WorkerCount      int
	SweeperInterval  time.Duration
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

	sweeperInterval := os.Getenv("SWEEPER_INTERVAL")
	if sweeperInterval == "" {
		sweeperInterval = "10s"
	}

	sweeperIntervalDuration, err := time.ParseDuration(sweeperInterval)
	if err != nil {
		sweeperIntervalDuration = 10 * time.Second
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
		SweeperInterval:  sweeperIntervalDuration,
	}
}

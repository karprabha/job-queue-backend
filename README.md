# WorkStream

A production-ready, concurrent job queue backend built in Go. Process jobs reliably with automatic retries, graceful shutdown, recovery semantics, and comprehensive observability.

## Description

WorkStream is a robust job queue system designed for handling asynchronous task processing. It provides a clean HTTP API for job submission and management, with built-in support for:

- **Concurrent Processing**: Worker pools process jobs in parallel
- **State Management**: Jobs transition through `pending` → `processing` → `completed`/`failed` states
- **Automatic Retries**: Failed jobs are automatically retried up to a configurable limit
- **Recovery**: On startup, recovers jobs that were in-flight during previous shutdowns
- **Backpressure**: Queue capacity limits prevent memory exhaustion
- **Observability**: Built-in metrics and structured logging
- **Graceful Shutdown**: Ensures in-flight jobs complete before termination

## Motivation

This project was built to learn and demonstrate production-grade backend engineering in Go. It showcases:

- Concurrency patterns (goroutines, channels, worker pools)
- State machine design for job lifecycle management
- Failure handling and retry logic
- Backpressure mechanisms
- Graceful shutdown coordination
- Recovery semantics for crash resilience
- Structured logging and metrics collection
- Testability and clean architecture

## Quick Start

### Prerequisites

- Go 1.25 or later
- Git

### Installation

```bash
git clone https://github.com/karprabha/job-queue-backend.git
cd job-queue-backend
go mod download
```

### Running the Server

```bash
go run cmd/server/main.go
```

The server will start on port `8080` by default. Configure via environment variables:

```bash
PORT=8080                    # Server port (default: 8080)
WORKER_COUNT=10              # Number of worker goroutines (default: 10)
JOB_QUEUE_CAPACITY=100       # Maximum queued jobs (default: 100)
SWEEPER_INTERVAL=10s         # Interval for retry sweeper (default: 10s)
```

## Usage

### Create a Job

Submit a new job for processing:

```bash
curl -X POST http://localhost:8080/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "type": "email_send",
    "payload": {
      "to": "user@example.com",
      "subject": "Welcome",
      "body": "Hello!"
    }
  }'
```

Response:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "email_send",
  "status": "pending",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### List All Jobs

Retrieve all jobs and their current status:

```bash
curl http://localhost:8080/jobs
```

### Get Metrics

View system metrics:

```bash
curl http://localhost:8080/metrics
```

Response includes:

- Total jobs created
- Jobs completed
- Jobs failed
- Current queue size

### Health Check

Check server health:

```bash
curl http://localhost:8080/health
```

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development

Run tests:

```bash
go test ./...
```

Build the binary:

```bash
go build -o workstream cmd/server/main.go
```

## License

This project is open source and available under the MIT License.

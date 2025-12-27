package ratelimiter

import (
	"context"
	"time"
)

type BurstyLimiter struct {
	capacity int
	rate     time.Duration
	tokens   chan time.Time
}

func NewBurstyLimiter(capacity int, rate time.Duration) *BurstyLimiter {
	tokens := make(chan time.Time, capacity)

	// Fill initial burst
	for i := 0; i < capacity; i++ {
		tokens <- time.Now()
	}

	// Refill tokens at the specified rate
	go func() {
		for t := range time.Tick(rate) {
			tokens <- t
		}
	}()

	return &BurstyLimiter{
		capacity: capacity,
		rate:     rate,
		tokens:   tokens,
	}
}

// Take waits for a token
func (b *BurstyLimiter) Take(ctx context.Context) error {
	select {
	case <-b.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

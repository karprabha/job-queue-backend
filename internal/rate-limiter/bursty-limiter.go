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

func NewBurstyLimiter(ctx context.Context, capacity int, rate time.Duration) *BurstyLimiter {
	tokens := make(chan time.Time, capacity)

	// Fill initial burst
	for i := 0; i < capacity; i++ {
		tokens <- time.Now()
	}

	ticker := time.NewTicker(rate)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case t := <-ticker.C:
				select {
				case tokens <- t: // try to add token
				default: // channel full, skip
				}
			case <-ctx.Done():
				return // exit goroutine if context is canceled
			}
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

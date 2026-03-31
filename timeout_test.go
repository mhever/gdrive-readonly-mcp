package main

import (
	"context"
	"testing"
	"time"
)

func TestWithTimeout(t *testing.T) {
	before := time.Now()
	ctx, cancel := withTimeout(context.Background())
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("withTimeout did not set a deadline")
	}

	// The deadline should be approximately apiTimeout from now.
	expected := before.Add(apiTimeout)
	tolerance := 1 * time.Second
	if deadline.Before(expected.Add(-tolerance)) || deadline.After(expected.Add(tolerance)) {
		t.Errorf("deadline = %v, want approximately %v (tolerance %v)", deadline, expected, tolerance)
	}
}

func TestApiLimiterExists(t *testing.T) {
	if apiLimiter == nil {
		t.Fatal("apiLimiter is nil")
	}

	// Verify the configured rate: 5 requests/sec.
	if apiLimiter.Limit() != 5 {
		t.Errorf("apiLimiter.Limit() = %v, want 5", apiLimiter.Limit())
	}

	// Verify the configured burst: 10.
	if apiLimiter.Burst() != 10 {
		t.Errorf("apiLimiter.Burst() = %v, want 10", apiLimiter.Burst())
	}
}

func TestApiLimiterRateLimits(t *testing.T) {
	// Drain the burst allowance, then verify the next request is delayed.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	burst := apiLimiter.Burst()

	// Consume the entire burst allowance.
	for i := 0; i < burst; i++ {
		if err := apiLimiter.Wait(ctx); err != nil {
			t.Fatalf("burst request %d failed: %v", i, err)
		}
	}

	// The next request should be delayed by the rate limiter.
	start := time.Now()
	if err := apiLimiter.Wait(ctx); err != nil {
		t.Fatalf("post-burst request failed: %v", err)
	}
	elapsed := time.Since(start)

	// At 5 req/s, the minimum delay is 200ms. Use 100ms as a conservative
	// lower bound to avoid flaky failures from scheduling jitter.
	if elapsed < 100*time.Millisecond {
		t.Errorf("post-burst request completed in %v, expected at least 100ms delay from rate limiter", elapsed)
	}
}

func TestApiLimiterRejectsExpiredContext(t *testing.T) {
	// A context that is already cancelled should cause Wait to return
	// an error immediately rather than blocking.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	err := apiLimiter.Wait(ctx)
	if err == nil {
		t.Error("apiLimiter.Wait() should return error for cancelled context")
	}
}

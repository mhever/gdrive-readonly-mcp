package main

import (
	"context"
	"time"

	"golang.org/x/time/rate"
)

// apiTimeout is the maximum duration for any single Google API call.
// If an API call takes longer than this, the context is cancelled and the
// call returns an error rather than stalling the MCP server indefinitely.
const apiTimeout = 30 * time.Second

// withTimeout wraps the given context with apiTimeout.
func withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, apiTimeout)
}

// apiLimiter rate-limits outbound Google API requests to prevent
// quota exhaustion when LLMs retry aggressively.
// Allows 5 sustained requests per second with a burst of 10.
var apiLimiter = rate.NewLimiter(rate.Limit(5), 10)

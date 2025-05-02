package retry

import (
	"math"
	"math/rand"
	"time"
)

// Strategy defines a retry strategy for failed operations
type Strategy struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// InitialDelay is the delay for the first retry attempt
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration

	// Factor is the multiplier for the delay after each retry
	Factor float64

	// Jitter adds randomness to the delay to prevent thundering herd
	Jitter bool
}

// DefaultStrategy creates a default retry strategy with exponential backoff
func DefaultStrategy(maxRetries int, initialDelay, maxDelay time.Duration) *Strategy {
	return &Strategy{
		MaxRetries:   maxRetries,
		InitialDelay: initialDelay,
		MaxDelay:     maxDelay,
		Factor:       2.0,
		Jitter:       true,
	}
}

// CalculateNextDelay determines the delay before the next retry
func (s *Strategy) CalculateNextDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	// Calculate delay with exponential backoff
	delay := float64(s.InitialDelay) * math.Pow(s.Factor, float64(attempt-1))
	
	// Apply jitter (between 80% and 120% of computed delay)
	if s.Jitter {
		jitter := 0.8 + 0.4*rand.Float64()
		delay = delay * jitter
	}

	// Cap at maximum delay
	if delay > float64(s.MaxDelay) {
		delay = float64(s.MaxDelay)
	}

	return time.Duration(delay)
}

// ShouldRetry determines if another retry should be attempted
func (s *Strategy) ShouldRetry(attempt int) bool {
	return attempt < s.MaxRetries
}

// CalculateRetryTime returns the absolute time when the next retry should occur
func (s *Strategy) CalculateRetryTime(attempt int) time.Time {
	return time.Now().Add(s.CalculateNextDelay(attempt))
}

// IsRetriable determines if an error should be retried based on its type or content
func IsRetriable(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific error types that should be retried
	// For example, network errors, temporary server errors, rate limiting

	// Simple string-based checking (can be enhanced with proper error types)
	errStr := err.Error()
	retriablePatterns := []string{
		"connection reset",
		"connection refused",
		"timeout",
		"deadline exceeded",
		"temporary failure",
		"server error",
		"status 5", // 5xx server errors
		"rate limit",
		"too many requests",
	}

	for _, pattern := range retriablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// contains is a simple helper to check if a string contains a substring
func contains(s, substr string) bool {
	return s != "" && substr != "" && s != substr && s[0:len(s)][0:len(substr)] == substr
}

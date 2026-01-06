package scopeconfig

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Default retry settings
const (
	DefaultMaxRetries        = 3
	DefaultInitialBackoff    = 100 * time.Millisecond
	DefaultMaxBackoff        = 10 * time.Second
	DefaultBackoffMultiplier = 2.0
)

// RetryPolicy defines the retry behavior for failed gRPC calls.
type RetryPolicy struct {
	// MaxRetries is the maximum number of retry attempts.
	// 0 means no retries, -1 means unlimited retries (use with caution).
	MaxRetries int

	// InitialBackoff is the initial backoff duration before the first retry.
	InitialBackoff time.Duration

	// MaxBackoff is the maximum backoff duration between retries.
	MaxBackoff time.Duration

	// BackoffMultiplier is the multiplier for exponential backoff.
	// Each retry will wait InitialBackoff * (BackoffMultiplier ^ attemptNumber).
	BackoffMultiplier float64

	// RetryableStatusCodes is a list of gRPC status codes that should trigger a retry.
	// If nil, a default set of retryable codes will be used.
	RetryableStatusCodes []codes.Code

	// OnRetry is an optional callback that is invoked before each retry attempt.
	// It receives the attempt number (1-indexed) and the error that triggered the retry.
	OnRetry func(attempt int, err error)
}

// DefaultRetryPolicy returns a retry policy with sensible defaults.
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:           DefaultMaxRetries,
		InitialBackoff:       DefaultInitialBackoff,
		MaxBackoff:           DefaultMaxBackoff,
		BackoffMultiplier:    DefaultBackoffMultiplier,
		RetryableStatusCodes: defaultRetryableStatusCodes(),
	}
}

// defaultRetryableStatusCodes returns the default set of gRPC status codes
// that are considered retryable (transient failures).
func defaultRetryableStatusCodes() []codes.Code {
	return []codes.Code{
		codes.Unavailable,       // Server unavailable (temporary network issue)
		codes.DeadlineExceeded,  // Request timeout
		codes.ResourceExhausted, // Rate limiting or resource exhaustion
		codes.Aborted,           // Operation aborted (may succeed on retry)
		codes.Internal,          // Internal server error (may be transient)
	}
}

// isRetryable checks if an error is retryable based on the retry policy.
func (p *RetryPolicy) isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Get the gRPC status code
	st, ok := status.FromError(err)
	if !ok {
		// If it's not a gRPC error, don't retry
		return false
	}

	code := st.Code()

	// Check if this code is in the retryable list
	retryableCodes := p.RetryableStatusCodes
	if retryableCodes == nil {
		retryableCodes = defaultRetryableStatusCodes()
	}

	for _, retryableCode := range retryableCodes {
		if code == retryableCode {
			return true
		}
	}

	return false
}

// calculateBackoff calculates the backoff duration for a given attempt using exponential backoff.
func (p *RetryPolicy) calculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return p.InitialBackoff
	}

	// Calculate exponential backoff: InitialBackoff * (Multiplier ^ attempt)
	backoff := float64(p.InitialBackoff) * math.Pow(p.BackoffMultiplier, float64(attempt-1))

	// Cap at MaxBackoff
	if backoff > float64(p.MaxBackoff) {
		backoff = float64(p.MaxBackoff)
	}

	return time.Duration(backoff)
}

// Execute executes a function with retry logic based on the policy.
// The function will be retried if it returns a retryable error.
// The context is checked before each retry attempt.
func (p *RetryPolicy) Execute(ctx context.Context, operation func() error) error {
	var lastErr error

	maxAttempts := p.MaxRetries + 1 // Total attempts = retries + initial attempt
	if p.MaxRetries < 0 {
		maxAttempts = -1 // Unlimited retries
	}

	for attempt := 0; maxAttempts < 0 || attempt < maxAttempts; attempt++ {
		// Check if context is cancelled before attempting
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("context cancelled after %d attempts: %w (last error: %v)", attempt, ctx.Err(), lastErr)
			}
			return ctx.Err()
		default:
		}

		// Execute the operation
		err := operation()
		if err == nil {
			// Success!
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !p.isRetryable(err) {
			// Non-retryable error, return immediately
			return err
		}

		// Check if we've exhausted retry attempts
		if maxAttempts >= 0 && attempt >= p.MaxRetries {
			// No more retries left
			return fmt.Errorf("max retries (%d) exceeded: %w", p.MaxRetries, err)
		}

		// Calculate backoff duration
		backoff := p.calculateBackoff(attempt + 1)

		// Call the OnRetry callback if provided
		if p.OnRetry != nil {
			p.OnRetry(attempt+1, err)
		} else {
			// Default logging
			log.Printf("Retry attempt %d/%d after error (will wait %v): %v",
				attempt+1, p.MaxRetries, backoff, err)
		}

		// Wait for backoff duration (or until context is cancelled)
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during backoff: %w (last error: %v)", ctx.Err(), lastErr)
		case <-time.After(backoff):
			// Continue to next retry
		}
	}

	// Should never reach here, but just in case
	return lastErr
}

// ExecuteWithResult executes a function that returns a result and an error.
// This is a generic version of Execute for functions that return values.
func ExecuteWithResult[T any](
	ctx context.Context,
	policy *RetryPolicy,
	operation func() (T, error),
) (T, error) {
	var result T

	err := policy.Execute(ctx, func() error {
		var opErr error
		result, opErr = operation()
		return opErr
	})

	if err != nil {
		// Return zero value and error
		var zero T
		return zero, err
	}

	return result, nil
}

// retryableOperation wraps a gRPC call with retry logic.
// This is a helper function used internally by the client.
func retryableOperation[T any](
	ctx context.Context,
	policy *RetryPolicy,
	methodName string,
	operation func() (T, error),
) (T, error) {
	if policy == nil {
		// No retry policy, execute once
		return operation()
	}

	return ExecuteWithResult(ctx, policy, operation)
}

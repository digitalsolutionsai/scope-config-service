package scopeconfig

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()

	assert.Equal(t, DefaultMaxRetries, policy.MaxRetries)
	assert.Equal(t, DefaultInitialBackoff, policy.InitialBackoff)
	assert.Equal(t, DefaultMaxBackoff, policy.MaxBackoff)
	assert.Equal(t, DefaultBackoffMultiplier, policy.BackoffMultiplier)
	assert.NotNil(t, policy.RetryableStatusCodes)
}

func TestRetryPolicy_IsRetryable(t *testing.T) {
	policy := DefaultRetryPolicy()

	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "non-gRPC error",
			err:       errors.New("generic error"),
			retryable: false,
		},
		{
			name:      "Unavailable - retryable",
			err:       status.Error(codes.Unavailable, "service unavailable"),
			retryable: true,
		},
		{
			name:      "DeadlineExceeded - retryable",
			err:       status.Error(codes.DeadlineExceeded, "deadline exceeded"),
			retryable: true,
		},
		{
			name:      "ResourceExhausted - retryable",
			err:       status.Error(codes.ResourceExhausted, "resource exhausted"),
			retryable: true,
		},
		{
			name:      "Internal - retryable",
			err:       status.Error(codes.Internal, "internal error"),
			retryable: true,
		},
		{
			name:      "NotFound - not retryable",
			err:       status.Error(codes.NotFound, "not found"),
			retryable: false,
		},
		{
			name:      "InvalidArgument - not retryable",
			err:       status.Error(codes.InvalidArgument, "invalid argument"),
			retryable: false,
		},
		{
			name:      "PermissionDenied - not retryable",
			err:       status.Error(codes.PermissionDenied, "permission denied"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := policy.isRetryable(tt.err)
			assert.Equal(t, tt.retryable, result)
		})
	}
}

func TestRetryPolicy_CalculateBackoff(t *testing.T) {
	policy := &RetryPolicy{
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        2 * time.Second,
		BackoffMultiplier: 2.0,
	}

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{
			name:     "attempt 0",
			attempt:  0,
			expected: 100 * time.Millisecond,
		},
		{
			name:     "attempt 1",
			attempt:  1,
			expected: 100 * time.Millisecond, // 100 * 2^0
		},
		{
			name:     "attempt 2",
			attempt:  2,
			expected: 200 * time.Millisecond, // 100 * 2^1
		},
		{
			name:     "attempt 3",
			attempt:  3,
			expected: 400 * time.Millisecond, // 100 * 2^2
		},
		{
			name:     "attempt 4",
			attempt:  4,
			expected: 800 * time.Millisecond, // 100 * 2^3
		},
		{
			name:     "attempt 5 - approaching MaxBackoff",
			attempt:  5,
			expected: 1600 * time.Millisecond, // 100 * 2^4 = 1600ms
		},
		{
			name:     "attempt 6 - capped at MaxBackoff",
			attempt:  6,
			expected: 2 * time.Second, // Would be 3200ms, but capped at 2s
		},
		{
			name:     "attempt 10 - still capped",
			attempt:  10,
			expected: 2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := policy.calculateBackoff(tt.attempt)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRetryPolicy_Execute_Success(t *testing.T) {
	policy := DefaultRetryPolicy()
	ctx := context.Background()

	callCount := 0
	operation := func() error {
		callCount++
		return nil // Immediate success
	}

	err := policy.Execute(ctx, operation)
	assert.NoError(t, err)
	assert.Equal(t, 1, callCount, "Should only call once on success")
}

func TestRetryPolicy_Execute_NonRetryableError(t *testing.T) {
	policy := DefaultRetryPolicy()
	ctx := context.Background()

	callCount := 0
	expectedErr := status.Error(codes.NotFound, "not found")
	operation := func() error {
		callCount++
		return expectedErr
	}

	err := policy.Execute(ctx, operation)
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, 1, callCount, "Should not retry non-retryable errors")
}

func TestRetryPolicy_Execute_RetryableError_EventualSuccess(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:        3,
		InitialBackoff:    1 * time.Millisecond, // Very short for testing
		MaxBackoff:        10 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}
	ctx := context.Background()

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 3 {
			return status.Error(codes.Unavailable, "unavailable")
		}
		return nil // Success on 3rd attempt
	}

	err := policy.Execute(ctx, operation)
	assert.NoError(t, err)
	assert.Equal(t, 3, callCount, "Should succeed on 3rd attempt")
}

func TestRetryPolicy_Execute_MaxRetriesExceeded(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:        2,
		InitialBackoff:    1 * time.Millisecond,
		MaxBackoff:        10 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}
	ctx := context.Background()

	callCount := 0
	expectedErr := status.Error(codes.Unavailable, "unavailable")
	operation := func() error {
		callCount++
		return expectedErr // Always fail
	}

	err := policy.Execute(ctx, operation)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max retries (2) exceeded")
	assert.Equal(t, 3, callCount, "Should call: initial + 2 retries")
}

func TestRetryPolicy_Execute_ContextCancelled(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:        5,
		InitialBackoff:    50 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
	}

	// Cancel context after short delay
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	callCount := 0
	operation := func() error {
		callCount++
		return status.Error(codes.Unavailable, "unavailable")
	}

	err := policy.Execute(ctx, operation)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context")
	// Should have made 1-3 attempts before context cancellation
	assert.GreaterOrEqual(t, callCount, 1)
	assert.LessOrEqual(t, callCount, 4)
}

func TestRetryPolicy_Execute_OnRetryCallback(t *testing.T) {
	var retryAttempts []int
	var retryErrors []error

	policy := &RetryPolicy{
		MaxRetries:        3,
		InitialBackoff:    1 * time.Millisecond,
		MaxBackoff:        10 * time.Millisecond,
		BackoffMultiplier: 2.0,
		OnRetry: func(attempt int, err error) {
			retryAttempts = append(retryAttempts, attempt)
			retryErrors = append(retryErrors, err)
		},
	}
	ctx := context.Background()

	callCount := 0
	expectedErr := status.Error(codes.Unavailable, "unavailable")
	operation := func() error {
		callCount++
		if callCount < 3 {
			return expectedErr
		}
		return nil // Success on 3rd attempt
	}

	err := policy.Execute(ctx, operation)
	assert.NoError(t, err)
	assert.Equal(t, 3, callCount)

	// Should have 2 retry callbacks (for attempts 1 and 2, before final success)
	assert.Len(t, retryAttempts, 2)
	assert.Equal(t, []int{1, 2}, retryAttempts)
	assert.Len(t, retryErrors, 2)
	for _, retryErr := range retryErrors {
		assert.Equal(t, expectedErr, retryErr)
	}
}

func TestExecuteWithResult_Success(t *testing.T) {
	policy := DefaultRetryPolicy()
	ctx := context.Background()

	operation := func() (string, error) {
		return "success", nil
	}

	result, err := ExecuteWithResult(ctx, policy, operation)
	assert.NoError(t, err)
	assert.Equal(t, "success", result)
}

func TestExecuteWithResult_Retry(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:        2,
		InitialBackoff:    1 * time.Millisecond,
		MaxBackoff:        10 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}
	ctx := context.Background()

	callCount := 0
	operation := func() (int, error) {
		callCount++
		if callCount < 3 {
			return 0, status.Error(codes.Unavailable, "unavailable")
		}
		return 42, nil
	}

	result, err := ExecuteWithResult(ctx, policy, operation)
	assert.NoError(t, err)
	assert.Equal(t, 42, result)
	assert.Equal(t, 3, callCount)
}

func TestExecuteWithResult_Error(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:        1,
		InitialBackoff:    1 * time.Millisecond,
		MaxBackoff:        10 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}
	ctx := context.Background()

	expectedErr := status.Error(codes.NotFound, "not found")
	operation := func() (string, error) {
		return "", expectedErr
	}

	result, err := ExecuteWithResult(ctx, policy, operation)
	assert.Error(t, err)
	assert.Equal(t, "", result) // Zero value
	assert.Equal(t, expectedErr, err)
}

func TestRetryableOperation_NoPolicy(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	operation := func() (string, error) {
		callCount++
		if callCount == 1 {
			return "", status.Error(codes.Unavailable, "unavailable")
		}
		return "success", nil
	}

	// No retry policy - should fail on first error
	result, err := retryableOperation(ctx, nil, "TestOp", operation)
	assert.Error(t, err)
	assert.Equal(t, "", result)
	assert.Equal(t, 1, callCount, "Should only call once without retry policy")
}

func TestRetryableOperation_WithPolicy(t *testing.T) {
	ctx := context.Background()
	policy := &RetryPolicy{
		MaxRetries:        2,
		InitialBackoff:    1 * time.Millisecond,
		MaxBackoff:        10 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	callCount := 0
	operation := func() (string, error) {
		callCount++
		if callCount < 2 {
			return "", status.Error(codes.Unavailable, "unavailable")
		}
		return "success", nil
	}

	result, err := retryableOperation(ctx, policy, "TestOp", operation)
	assert.NoError(t, err)
	assert.Equal(t, "success", result)
	assert.Equal(t, 2, callCount, "Should retry once and succeed")
}

func BenchmarkRetryPolicy_Execute_NoRetry(b *testing.B) {
	policy := DefaultRetryPolicy()
	ctx := context.Background()

	operation := func() error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = policy.Execute(ctx, operation)
	}
}

func BenchmarkRetryPolicy_Execute_WithRetries(b *testing.B) {
	policy := &RetryPolicy{
		MaxRetries:        3,
		InitialBackoff:    1 * time.Microsecond,
		MaxBackoff:        10 * time.Microsecond,
		BackoffMultiplier: 2.0,
	}
	ctx := context.Background()

	callCount := 0
	operation := func() error {
		callCount++
		if callCount%3 != 0 {
			return status.Error(codes.Unavailable, "unavailable")
		}
		callCount = 0
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = policy.Execute(ctx, operation)
	}
}

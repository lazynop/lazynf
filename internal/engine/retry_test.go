package engine

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

var _ net.Error = timeoutErr{}

func TestRetry_SucceedsFirstTry(t *testing.T) {
	calls := 0
	err := retry(context.Background(), func() error {
		calls++
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
}

func TestRetry_NonRetriableReturnsImmediately(t *testing.T) {
	calls := 0
	want := errors.New("404 not found")
	err := retry(context.Background(), func() error {
		calls++
		return want
	})
	require.Same(t, want, err)
	require.Equal(t, 1, calls)
}

func TestRetry_RetriableEventuallySucceeds(t *testing.T) {
	orig := retryDelays
	retryDelays = []time.Duration{0, 5 * time.Millisecond, 10 * time.Millisecond}
	t.Cleanup(func() { retryDelays = orig })

	calls := 0
	err := retry(context.Background(), func() error {
		calls++
		if calls < 3 {
			return timeoutErr{}
		}
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 3, calls)
}

func TestRetry_AllAttemptsFail(t *testing.T) {
	orig := retryDelays
	retryDelays = []time.Duration{0, 5 * time.Millisecond, 10 * time.Millisecond}
	t.Cleanup(func() { retryDelays = orig })

	calls := 0
	err := retry(context.Background(), func() error {
		calls++
		return timeoutErr{}
	})
	require.Error(t, err)
	require.IsType(t, timeoutErr{}, err)
	require.Equal(t, 3, calls)
}

func TestRetry_RespectsContextCancel(t *testing.T) {
	orig := retryDelays
	retryDelays = []time.Duration{0, 100 * time.Millisecond, 100 * time.Millisecond}
	t.Cleanup(func() { retryDelays = orig })

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	err := retry(ctx, func() error {
		calls++
		return timeoutErr{}
	})
	require.ErrorIs(t, err, context.Canceled)
	require.GreaterOrEqual(t, calls, 1)
}

func TestRetry_ReturnsActualErrorAfterRetriableSwap(t *testing.T) {
	orig := retryDelays
	retryDelays = []time.Duration{0, 5 * time.Millisecond, 10 * time.Millisecond}
	t.Cleanup(func() { retryDelays = orig })

	want := errors.New("not retriable")
	calls := 0
	err := retry(context.Background(), func() error {
		calls++
		return want
	})
	require.Same(t, want, err)
	require.Equal(t, 1, calls)
}

package engine

import (
	"context"
	"errors"
	"net"
	"time"
)

// retryDelays is the backoff sequence: 1s, 2s, 4s. Maximum of 3 attempts.
var retryDelays = []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}

// isRetriableNetErr reports whether err is a transient network error worth
// retrying. Currently conservative: only matches net.Error implementations
// whose Timeout() returns true. Explicitly NOT retried: HTTP non-2xx
// (server-definitive), connection refused (not a net.Error.Timeout()),
// context cancellation/deadline. Extending coverage to ECONNREFUSED or
// transient DNS errors is a future enhancement — keep it tight for now to
// avoid masking real failures.
func isRetriableNetErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var ne net.Error
	if errors.As(err, &ne) {
		return ne.Timeout()
	}
	return false
}

// retry runs fn up to 3 times (1 + 2 retries) with exponential backoff
// 1s/2s/4s. It returns immediately on ctx.Done(). Returns the last error
// observed if every attempt fails.
func retry(ctx context.Context, fn func() error) error {
	err := fn()
	if err == nil || !isRetriableNetErr(err) {
		return err
	}
	for _, d := range retryDelays[1:] {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
		}
		err = fn()
		if err == nil || !isRetriableNetErr(err) {
			return err
		}
	}
	return err
}

// retryCall runs fn up to 3 times with the same backoff as retry, returning
// the LAST error fn produced (whether retriable or not). If ctx fires during
// a backoff sleep, returns ctx.Err() instead. Use this when the caller wants
// the call's actual error surfaced — unlike retry, retryable errors are
// transparently retried but non-retriable errors are returned immediately.
func retryCall(ctx context.Context, fn func() error) error {
	err := fn()
	if err == nil || !isRetriableNetErr(err) {
		return err
	}
	for _, d := range retryDelays[1:] {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
		}
		err = fn()
		if err == nil || !isRetriableNetErr(err) {
			return err
		}
	}
	return err
}

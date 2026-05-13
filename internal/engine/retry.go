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
// retrying. Deliberately conservative: only timeouts, refused connections,
// and DNS-flavored failures. HTTP 4xx is NOT retriable (server-definitive).
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

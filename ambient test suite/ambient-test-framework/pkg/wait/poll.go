// pkg/wait/poll.go
package wait

import (
	"context"
	"fmt"
	"time"
)

// ConditionFunc is a function that returns true when a condition is met.
type ConditionFunc func(ctx context.Context) (bool, error)

// Until polls fn every interval until it returns (true, nil) or the context
// expires or fn returns an error.
func Until(ctx context.Context,
	fn ConditionFunc,
	interval, timeout time.Duration) error {

	deadline := time.Now().Add(timeout)
	for {
		done, err := fn(ctx)
		if err != nil {
			return fmt.Errorf("condition error: %w", err)
		}
		if done {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out after %s waiting for condition", timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

// UntilNoError polls fn every interval until it returns nil or the context expires.
func UntilNoError(ctx context.Context,
	fn func(ctx context.Context) error,
	interval, timeout time.Duration) error {

	return Until(ctx, func(ctx context.Context) (bool, error) {
		if err := fn(ctx); err != nil {
			return false, nil
		}
		return true, nil
	}, interval, timeout)
}

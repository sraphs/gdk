// Package retry provides retry logic.
package retry

import (
	"context"
	"fmt"
	"time"

	"github.com/googleapis/gax-go/v2"
)

// Call calls the supplied function f repeatedly, using the isRetryable function and
// the provided backoff parameters to control the repetition.
//
// When f returns nil, Call immediately returns nil.
//
// When f returns an error for which isRetryable returns false, Call immediately
// returns that error.
//
// When f returns an error for which isRetryable returns true, Call sleeps for the
// provided backoff value and invokes f again.
//
// When the provided context is done, Retry returns a ContextError that includes both
// ctx.Error() and the last error returned by f, or nil if there isn't one.
func Call(ctx context.Context, bo gax.Backoff, isRetryable func(error) bool, f func() error) error {
	return call(ctx, bo, isRetryable, f, gax.Sleep)
}

// Split out for testing.
func call(ctx context.Context, bo gax.Backoff, isRetryable func(error) bool, f func() error,
	sleep func(context.Context, time.Duration) error) error {
	// Do nothing if context is done on entry.
	if err := ctx.Err(); err != nil {
		return &ContextError{CtxErr: err}
	}
	for {
		err := f()
		if err == nil {
			return nil
		}
		if !isRetryable(err) {
			return err
		}
		if cerr := sleep(ctx, bo.Pause()); cerr != nil {
			return &ContextError{CtxErr: cerr, FuncErr: err}
		}
	}
}

// A ContextError contains both a context error (either context.Canceled or
// context.DeadlineExceeded), and the last error from the function being retried,
// or nil if the function was never called.
type ContextError struct {
	CtxErr  error // The error obtained from ctx.Err()
	FuncErr error // The error obtained from the function being retried, or nil
}

func (e *ContextError) Error() string {
	return fmt.Sprintf("%v; last error: %v", e.CtxErr, e.FuncErr)
}

// Is returns true iff one of the two errors held in e is equal to target.
func (e *ContextError) Is(target error) bool {
	return e.CtxErr == target || e.FuncErr == target
}

package tui

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCollectWithTimeoutReturnsDeadline(t *testing.T) {
	_, err := collectWithTimeout(context.Background(), 10*time.Millisecond, func(ctx context.Context) (int, error) {
		<-ctx.Done()
		return 0, ctx.Err()
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("collectWithTimeout() error = %v, want context deadline exceeded", err)
	}
}

func TestCollectWithTimeoutHonorsParentCancellation(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collectWithTimeout(parent, time.Second, func(ctx context.Context) (int, error) {
		<-ctx.Done()
		return 0, ctx.Err()
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("collectWithTimeout() error = %v, want context canceled", err)
	}
}

package auth

import (
	"context"
	"testing"
	"time"
)

func TestRunSessionGCCancels(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- RunSessionGC(ctx)
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunSessionGC returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("RunSessionGC did not stop after context cancellation")
	}
}

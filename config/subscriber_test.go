package config

import (
	"sync/atomic"
	"testing"
)

func TestConfigSubscriberUnsubscribeAllIsIdempotent(t *testing.T) {
	var subscriber ConfigSubscriber
	var calls atomic.Int32

	subscriber.Add(func() {
		calls.Add(1)
	})
	subscriber.Add(func() {
		calls.Add(1)
	})

	subscriber.UnsubscribeAll()
	subscriber.UnsubscribeAll()

	if got := calls.Load(); got != 2 {
		t.Fatalf("expected each unsubscribe function to run once, got %d calls", got)
	}
}

func TestConfigSubscriberAddAfterUnsubscribeRunsImmediately(t *testing.T) {
	var subscriber ConfigSubscriber
	var calls atomic.Int32

	subscriber.UnsubscribeAll()
	subscriber.Add(func() {
		calls.Add(1)
	})

	if got := calls.Load(); got != 1 {
		t.Fatalf("expected late unsubscribe function to run immediately, got %d calls", got)
	}
}

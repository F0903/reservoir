package auth

import (
	"context"
	"reservoir/utils/syncmap"
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

func TestDestroySessionsForUserExceptKeepsCurrentSession(t *testing.T) {
	resetSessionsForTest()
	defer resetSessionsForTest()

	current := CreateSession(1)
	other := CreateSession(1)
	differentUser := CreateSession(2)

	deleted := DestroySessionsForUserExcept(1, current.ID)
	if deleted != 1 {
		t.Fatalf("expected one session to be destroyed, got %d", deleted)
	}

	if _, ok := GetSession(current.ID); !ok {
		t.Fatal("expected current session to remain")
	}
	if _, ok := GetSession(other.ID); ok {
		t.Fatal("expected other user session to be destroyed")
	}
	if _, ok := GetSession(differentUser.ID); !ok {
		t.Fatal("expected different user's session to remain")
	}
}

func resetSessionsForTest() {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	sessionStore = syncmap.New[string, *Session]()
	gcRunning = false
}

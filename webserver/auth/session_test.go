package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionManagerRunGCCancels(t *testing.T) {
	manager := NewSessionManager()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- manager.RunGC(ctx)
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunGC returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("RunGC did not stop after context cancellation")
	}
}

func TestSessionManagerDestroySessionsForUserExceptKeepsCurrentSession(t *testing.T) {
	manager := NewSessionManager()

	current := manager.Create(1)
	other := manager.Create(1)
	differentUser := manager.Create(2)

	deleted := manager.DestroySessionsForUserExcept(1, current.ID)
	if deleted != 1 {
		t.Fatalf("expected one session to be destroyed, got %d", deleted)
	}

	if _, ok := manager.Get(current.ID); !ok {
		t.Fatal("expected current session to remain")
	}
	if _, ok := manager.Get(other.ID); ok {
		t.Fatal("expected other user session to be destroyed")
	}
	if _, ok := manager.Get(differentUser.ID); !ok {
		t.Fatal("expected different user's session to remain")
	}
}

func TestSessionManagersKeepIndependentStores(t *testing.T) {
	first := NewSessionManager()
	second := NewSessionManager()
	session := first.Create(1)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(session.BuildSessionCookie())

	if _, ok := second.SessionFromRequest(req); ok {
		t.Fatal("expected second manager not to find first manager's session")
	}

	got, ok := first.SessionFromRequest(req)
	if !ok {
		t.Fatal("expected first manager to find its session")
	}
	if got.UserID != session.UserID {
		t.Fatalf("expected user id %d, got %d", session.UserID, got.UserID)
	}
}

func TestSessionManagerGetRejectsExpiredSessions(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	manager := NewSessionManager()
	manager.now = func() time.Time {
		return now
	}

	session := manager.Create(1)
	now = now.Add(defaultLifetime + time.Second)

	if _, ok := manager.Get(session.ID); ok {
		t.Fatal("expected expired session to be rejected")
	}
	if _, ok := manager.store.Get(session.ID); ok {
		t.Fatal("expected expired session to be removed")
	}
}

package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
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
	if _, ok := manager.store[session.ID]; ok {
		t.Fatal("expected expired session to be removed")
	}
}

func TestSessionManagerConcurrentGetExtendsSession(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	manager := NewSessionManager()
	manager.lifetime = time.Minute
	manager.now = func() time.Time {
		return now
	}

	session := manager.Create(1)
	expectedExpiry := now.Add(time.Minute)
	const goroutines = 16
	const iterations = 200

	var wg sync.WaitGroup
	errs := make(chan string, goroutines)
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				got, ok := manager.Get(session.ID)
				if !ok {
					errs <- "expected session to remain available"
					return
				}
				if !got.ExpiresAt.Equal(expectedExpiry) {
					errs <- fmt.Sprintf("expected expiry %s, got %s", expectedExpiry, got.ExpiresAt)
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errs)

	if err := <-errs; err != "" {
		t.Fatal(err)
	}
}

func TestSessionManagerConcurrentMutations(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	manager := NewSessionManager()
	manager.now = func() time.Time {
		return now
	}
	var nextID atomic.Int64
	manager.newID = func() string {
		return fmt.Sprintf("sid-%d", nextID.Add(1))
	}

	const goroutines = 16
	const iterations = 100

	var wg sync.WaitGroup
	errs := make(chan string, goroutines)
	for worker := range goroutines {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			userID := int64(worker + 1)
			for range iterations {
				session := manager.Create(userID)
				if _, ok := manager.Get(session.ID); !ok {
					errs <- fmt.Sprintf("expected created session %q to be available", session.ID)
					return
				}
				manager.DestroySessionsForUserExcept(userID, session.ID)
				if _, ok := manager.Get(session.ID); !ok {
					errs <- fmt.Sprintf("expected kept session %q to remain available", session.ID)
					return
				}
				manager.Destroy(session)
				if _, ok := manager.Get(session.ID); ok {
					errs <- fmt.Sprintf("expected destroyed session %q to be unavailable", session.ID)
					return
				}
			}
		}(worker)
	}

	wg.Wait()
	close(errs)

	if err := <-errs; err != "" {
		t.Fatal(err)
	}
}

package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoginAttemptLimiterBlocksAfterFailures(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	limiter := NewLoginAttemptLimiter()
	limiter.limit = 2
	limiter.window = time.Minute
	limiter.block = 5 * time.Minute
	limiter.now = func() time.Time {
		return now
	}

	if _, blocked := limiter.Check("203.0.113.10", "Admin"); blocked {
		t.Fatal("expected fresh login key not to be blocked")
	}

	limiter.RecordFailure("203.0.113.10", "Admin")
	if _, blocked := limiter.Check("203.0.113.10", "admin"); blocked {
		t.Fatal("expected one failure not to block")
	}

	limiter.RecordFailure("203.0.113.10", "admin")
	retryAfter, blocked := limiter.Check("203.0.113.10", " admin ")
	if !blocked {
		t.Fatal("expected repeated failures to block")
	}
	if retryAfter != limiter.block {
		t.Fatalf("expected retryAfter %s, got %s", limiter.block, retryAfter)
	}
}

func TestLoginAttemptLimiterSuccessClearsFailures(t *testing.T) {
	limiter := NewLoginAttemptLimiter()
	limiter.limit = 1

	limiter.RecordFailure("203.0.113.10", "admin")
	if _, blocked := limiter.Check("203.0.113.10", "admin"); !blocked {
		t.Fatal("expected login key to be blocked")
	}

	limiter.RecordSuccess("203.0.113.10", "admin")
	if _, blocked := limiter.Check("203.0.113.10", "admin"); blocked {
		t.Fatal("expected successful login to clear failed attempts")
	}
}

func TestClientAddressUsesRemoteHost(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.10:54321"

	if got := ClientAddress(req, nil); got != "203.0.113.10" {
		t.Fatalf("expected remote host, got %q", got)
	}
}

package auth

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSessionIDsAreNotLogged(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })

	manager := NewSessionManager()
	manager.extendThreshold = manager.lifetime * 2 // Makes Get take its "extend" branch.

	session := manager.Create(1)
	if _, ok := manager.Get(session.ID); !ok {
		t.Fatal("expected session to exist")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: session.ID})
	if _, ok := manager.SessionFromRequest(req); !ok {
		t.Fatal("expected session from cookie")
	}
	manager.Destroy(session)

	expiring := manager.Create(2)
	manager.deleteExpired(time.Now().Add(2 * manager.lifetime))

	for _, id := range []string{session.ID, expiring.ID} {
		if strings.Contains(logs.String(), id) {
			t.Fatalf("session ID %q appears in logs:\n%s", id, logs.String())
		}
	}
}

func TestClientAddressOnlyBelievesTrustedProxies(t *testing.T) {
	trusted := []netip.Prefix{
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("192.168.1.1/32"),
	}

	tests := []struct {
		name         string
		remote       string
		forwardedFor []string
		want         string
	}{
		{"direct client", "203.0.113.10:1234", nil, "203.0.113.10"},
		{"untrusted peer cannot spoof", "203.0.113.10:1234", []string{"198.51.100.7"}, "203.0.113.10"},
		{"trusted peer with no header", "10.0.0.5:1234", nil, "10.0.0.5"},
		{"trusted peer names client", "10.0.0.5:1234", []string{"198.51.100.7"}, "198.51.100.7"},
		{"client spoofs left of real entry", "10.0.0.5:1234", []string{"1.1.1.1, 198.51.100.7"}, "198.51.100.7"},
		{"chain of trusted proxies", "10.0.0.5:1234", []string{"198.51.100.7, 192.168.1.1, 10.0.0.9"}, "198.51.100.7"},
		{"separate header lines", "10.0.0.5:1234", []string{"198.51.100.7", "10.0.0.9"}, "198.51.100.7"},
		{"garbage stops the walk", "10.0.0.5:1234", []string{"198.51.100.7, garbage, 10.0.0.9"}, "10.0.0.9"},
		{"only garbage keeps the peer", "10.0.0.5:1234", []string{"garbage"}, "10.0.0.5"},
		{"mapped ipv6 entry", "10.0.0.5:1234", []string{"::ffff:198.51.100.7"}, "198.51.100.7"},
		{"mapped ipv6 peer", "[::ffff:10.0.0.5]:1234", []string{"198.51.100.7"}, "198.51.100.7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/login", nil)
			req.RemoteAddr = tt.remote
			for _, value := range tt.forwardedFor {
				req.Header.Add("X-Forwarded-For", value)
			}

			if got := ClientAddress(req, trusted); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoginAttemptLimiterBlocksAddressAcrossUsernames(t *testing.T) {
	limiter := NewLoginAttemptLimiter()
	limiter.ipLimit = 3

	for i := range 3 {
		limiter.RecordFailure("203.0.113.10", fmt.Sprint("random-", i))
	}

	if _, blocked := limiter.Check("203.0.113.10", "never-tried"); !blocked {
		t.Fatal("expected the address to be blocked for every username")
	}
	if _, blocked := limiter.Check("203.0.113.99", "never-tried"); blocked {
		t.Fatal("expected other addresses to be unaffected")
	}
}

func TestLoginAttemptLimiterSuccessKeepsAddressCount(t *testing.T) {
	limiter := NewLoginAttemptLimiter()
	limiter.ipLimit = 2

	limiter.RecordFailure("203.0.113.10", "a")
	limiter.RecordSuccess("203.0.113.10", "attacker-owned-account")
	limiter.RecordFailure("203.0.113.10", "b")

	if _, blocked := limiter.Check("203.0.113.10", "c"); !blocked {
		t.Fatal("expected a successful login not to reset the address count")
	}
}

func TestLoginAttemptLimiterSweepsOldRecords(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	limiter := NewLoginAttemptLimiter()
	limiter.now = func() time.Time { return now }

	for i := range 20 {
		limiter.RecordFailure("203.0.113.10", fmt.Sprint("user-", i))
	}

	now = now.Add(limiter.window + time.Second)
	limiter.RecordFailure("203.0.113.11", "fresh")

	if got := len(limiter.attempts); got != 2 {
		t.Fatalf("expected only the fresh user and address records, got %d", got)
	}
}

func TestAuthenticateLimitsConcurrentHashing(t *testing.T) {
	store := newTestBootstrapUserStore(t)

	for range cap(hashingSlots) {
		hashingSlots <- struct{}{}
	}
	releaseSlots := sync.OnceFunc(func() {
		for range cap(hashingSlots) {
			<-hashingSlots
		}
	})
	t.Cleanup(releaseSlots)

	done := make(chan struct{})
	go func() {
		defer close(done)
		creds := Credentials{Username: "nobody", Password: "wrong"}
		creds.Authenticate(store)
	}()

	select {
	case <-done:
		t.Fatal("expected hashing to wait for a free slot")
	case <-time.After(200 * time.Millisecond):
	}

	releaseSlots()
	<-done
}

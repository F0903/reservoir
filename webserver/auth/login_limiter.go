package auth

import (
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	defaultLoginAttemptLimit   = 5
	defaultLoginAttemptIPLimit = 50 // High, because users behind NAT or an untrusted proxy share one address.
	defaultLoginAttemptWindow  = 15 * time.Minute
	defaultLoginAttemptBlock   = 15 * time.Minute
)

type loginAttemptKey struct {
	remote   string
	username string
	wholeIP  bool // Counts every failure from the address, whatever the username.
}

type loginAttemptRecord struct {
	count        int
	firstFailure time.Time
	blockedUntil time.Time
}

type LoginAttemptLimiter struct {
	mu        sync.Mutex
	attempts  map[loginAttemptKey]loginAttemptRecord
	limit     int
	ipLimit   int
	window    time.Duration
	block     time.Duration
	now       func() time.Time
	lastSweep time.Time
}

func NewLoginAttemptLimiter() *LoginAttemptLimiter {
	return &LoginAttemptLimiter{
		attempts: make(map[loginAttemptKey]loginAttemptRecord),
		limit:    defaultLoginAttemptLimit,
		ipLimit:  defaultLoginAttemptIPLimit,
		window:   defaultLoginAttemptWindow,
		block:    defaultLoginAttemptBlock,
		now:      time.Now,
	}
}

func (l *LoginAttemptLimiter) Check(remote string, username string) (time.Duration, bool) {
	if l == nil {
		return 0, false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	key := normalizeLoginAttemptKey(remote, username)
	blockedUntil := l.attempts[key].blockedUntil
	if ipBlockedUntil := l.attempts[wholeIPKey(key)].blockedUntil; ipBlockedUntil.After(blockedUntil) {
		blockedUntil = ipBlockedUntil
	}

	if !blockedUntil.After(now) {
		return 0, false
	}
	return blockedUntil.Sub(now), true
}

func (l *LoginAttemptLimiter) RecordFailure(remote string, username string) {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.lastSweep) > l.window {
		l.sweep(now)
		l.lastSweep = now
	}

	key := normalizeLoginAttemptKey(remote, username)
	l.recordFailure(key, l.limit, now)
	l.recordFailure(wholeIPKey(key), l.ipLimit, now)
}

func (l *LoginAttemptLimiter) recordFailure(key loginAttemptKey, limit int, now time.Time) {
	record := l.attempts[key]
	if record.firstFailure.IsZero() || now.Sub(record.firstFailure) > l.window {
		record = loginAttemptRecord{firstFailure: now}
	}

	record.count++
	if record.count >= limit {
		record.blockedUntil = now.Add(l.block)
	}
	l.attempts[key] = record
}

// sweep drops records that can no longer affect a login, so the map can't grow forever.
func (l *LoginAttemptLimiter) sweep(now time.Time) {
	for key, record := range l.attempts {
		if now.Sub(record.firstFailure) > l.window && !record.blockedUntil.After(now) {
			delete(l.attempts, key)
		}
	}
}

func (l *LoginAttemptLimiter) RecordSuccess(remote string, username string) {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.attempts, normalizeLoginAttemptKey(remote, username))
}

func normalizeLoginAttemptKey(remote string, username string) loginAttemptKey {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		remote = "unknown"
	}

	return loginAttemptKey{
		remote:   remote,
		username: strings.ToLower(strings.TrimSpace(username)),
	}
}

func wholeIPKey(key loginAttemptKey) loginAttemptKey {
	return loginAttemptKey{remote: key.remote, wholeIP: true}
}

// ClientAddress returns the address to apply login limits to. X-Forwarded-For is only
// believed from a trusted proxy, and is read right to left because each proxy appends
// to the end: anything further left could have been written by the client itself.
func ClientAddress(r *http.Request, trustedProxies []netip.Prefix) string {
	if r == nil {
		return ""
	}

	remote := strings.TrimSpace(r.RemoteAddr)
	if host, _, err := net.SplitHostPort(remote); err == nil {
		remote = host
	}

	peer, err := netip.ParseAddr(remote)
	if err != nil || !isTrustedProxy(peer, trustedProxies) {
		return remote
	}

	client := peer.Unmap().String()
	hops := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
	for _, hop := range slices.Backward(hops) {
		hop, err := netip.ParseAddr(strings.TrimSpace(hop))
		if err != nil {
			break
		}
		client = hop.Unmap().WithZone("").String()
		if !isTrustedProxy(hop, trustedProxies) {
			break
		}
	}
	return client
}

func isTrustedProxy(addr netip.Addr, trustedProxies []netip.Prefix) bool {
	addr = addr.Unmap().WithZone("")
	return slices.ContainsFunc(trustedProxies, func(prefix netip.Prefix) bool {
		return prefix.Contains(addr)
	})
}

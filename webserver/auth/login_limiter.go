package auth

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	defaultLoginAttemptLimit  = 5
	defaultLoginAttemptWindow = 15 * time.Minute
	defaultLoginAttemptBlock  = 15 * time.Minute
)

type loginAttemptKey struct {
	remote   string
	username string
}

type loginAttemptRecord struct {
	count        int
	firstFailure time.Time
	blockedUntil time.Time
}

type LoginAttemptLimiter struct {
	mu       sync.Mutex
	attempts map[loginAttemptKey]loginAttemptRecord
	limit    int
	window   time.Duration
	block    time.Duration
	now      func() time.Time
}

func NewLoginAttemptLimiter() *LoginAttemptLimiter {
	return &LoginAttemptLimiter{
		attempts: make(map[loginAttemptKey]loginAttemptRecord),
		limit:    defaultLoginAttemptLimit,
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
	record, ok := l.attempts[key]
	if !ok {
		return 0, false
	}
	if !record.blockedUntil.After(now) {
		return 0, false
	}
	return record.blockedUntil.Sub(now), true
}

func (l *LoginAttemptLimiter) RecordFailure(remote string, username string) {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	key := normalizeLoginAttemptKey(remote, username)
	record := l.attempts[key]
	if record.firstFailure.IsZero() || now.Sub(record.firstFailure) > l.window {
		record = loginAttemptRecord{firstFailure: now}
	}

	record.count++
	if record.count >= l.limit {
		record.blockedUntil = now.Add(l.block)
	}
	l.attempts[key] = record
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

func ClientAddress(r *http.Request) string {
	if r == nil {
		return ""
	}

	remote := strings.TrimSpace(r.RemoteAddr)
	host, _, err := net.SplitHostPort(remote)
	if err == nil {
		return host
	}
	return remote
}

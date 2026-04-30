package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"log/slog"
	"net/http"
	"reservoir/utils/syncmap"
	"sync"
	"time"
)

var ErrNoSession = errors.New("no user session found")

const (
	sessionCookieName = "reservoir.sid"
	defaultLifetime   = 1 * time.Hour
	extendThreshold   = 10 * time.Minute
	gcInterval        = 15 * time.Minute
)

type Session struct {
	manager              *SessionManager
	ID                   string
	UserID               int64
	CreatedAt, ExpiresAt time.Time
}

type SessionManager struct {
	store           *syncmap.SyncMap[string, *Session]
	mu              sync.Mutex
	gcRunning       bool
	lifetime        time.Duration
	extendThreshold time.Duration
	gcInterval      time.Duration
	now             func() time.Time
	newID           func() string
}

var defaultSessionManager = NewSessionManager()

func NewSessionManager() *SessionManager {
	return &SessionManager{
		store:           syncmap.New[string, *Session](),
		lifetime:        defaultLifetime,
		extendThreshold: extendThreshold,
		gcInterval:      gcInterval,
		now:             time.Now,
		newID:           rand.Text,
	}
}

func DefaultSessionManager() *SessionManager {
	return defaultSessionManager
}

func (m *SessionManager) RunGC(ctx context.Context) error {
	m.mu.Lock()
	if m.gcRunning {
		m.mu.Unlock()
		return nil
	}
	m.gcRunning = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.gcRunning = false
		m.mu.Unlock()
	}()

	slog.Info("Starting session garbage collector", "interval", m.gcInterval.String(), "session_lifetime", m.lifetime.String())
	ticker := time.NewTicker(m.gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.deleteExpired(m.now())
		case <-ctx.Done():
			slog.Info("Session garbage collector stopped")
			return nil
		}
	}
}

func (m *SessionManager) StartGC(ctx context.Context) {
	go func() {
		if err := m.RunGC(ctx); err != nil {
			slog.Error("Session garbage collector stopped with error", "error", err)
		}
	}()
}

func (m *SessionManager) Get(sid string) (*Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, ok := m.store.Get(sid)
	if !ok {
		return nil, false
	}

	now := m.now()
	if !sess.ExpiresAt.After(now) {
		m.store.Delete(sid)
		return nil, false
	}

	if sess.ExpiresAt.Sub(now) <= m.extendThreshold {
		slog.Debug("Session close to expiring, extending expiration", "session_id", sid, "expires_at", sess.ExpiresAt)
		sess.ExpiresAt = now.Add(m.lifetime)
		m.store.Set(sid, sess)
	}

	sessCopy := *sess
	return &sessCopy, true
}

func (m *SessionManager) Create(userID int64) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	slog.Debug("Creating new user session...", "user_id", userID)
	sid := m.newID()
	now := m.now()
	sess := &Session{
		manager:   m,
		ID:        sid,
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(m.lifetime),
	}

	m.store.Set(sid, sess)
	slog.Debug("Created new session", "session_id", sid, "expires_at", sess.ExpiresAt)

	sessCopy := *sess
	return &sessCopy
}

func (m *SessionManager) DestroySessionsForUserExcept(userID int64, keepSessionID string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	deleted := 0
	for item := range m.store.Items() {
		if item.UserID != userID || item.ID == keepSessionID {
			continue
		}
		m.store.Delete(item.ID)
		deleted++
	}
	if deleted > 0 {
		slog.Debug("Destroyed user sessions", "user_id", userID, "deleted", deleted)
	}
	return deleted
}

func (m *SessionManager) SessionFromRequest(r *http.Request) (sess *Session, ok bool) {
	sid, err := r.Cookie(sessionCookieName)
	if errors.Is(err, http.ErrNoCookie) {
		return nil, false
	}
	if err != nil {
		return nil, false
	}

	slog.Debug("Getting session from cookie...", "session_id", sid.Value)
	sess, ok = m.Get(sid.Value)
	if !ok {
		return nil, false
	}
	slog.Debug("Got session from cookie", "session_id", sid.Value, "expires_at", sess.ExpiresAt)

	return sess, true
}

func (m *SessionManager) Destroy(sess *Session) {
	if sess == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.store.Delete(sess.ID)
	slog.Debug("Destroyed session", "session_id", sess.ID)
}

func (m *SessionManager) deleteExpired(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for item := range m.store.Items() {
		if item.ExpiresAt.Before(now) {
			m.store.Delete(item.ID)
			slog.Debug("Deleted expired session", "session_id", item.ID)
		}
	}
}

func RunSessionGC(ctx context.Context) error {
	return defaultSessionManager.RunGC(ctx)
}

func StartSessionGC(ctx context.Context) {
	defaultSessionManager.StartGC(ctx)
}

func GetSession(sid string) (*Session, bool) {
	return defaultSessionManager.Get(sid)
}

func CreateSession(userID int64) *Session {
	return defaultSessionManager.Create(userID)
}

func DestroySessionsForUserExcept(userID int64, keepSessionID string) int {
	return defaultSessionManager.DestroySessionsForUserExcept(userID, keepSessionID)
}

func SessionFromRequest(r *http.Request) (sess *Session, ok bool) {
	return defaultSessionManager.SessionFromRequest(r)
}

func (s *Session) BuildSessionCookie() *http.Cookie {
	return &http.Cookie{
		Path:     "/",
		Name:     sessionCookieName,
		Value:    s.ID,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}

func (s *Session) Destroy() {
	if s.manager != nil {
		s.manager.Destroy(s)
		return
	}
	defaultSessionManager.Destroy(s)
}

package auth

import (
	"crypto/rand"
	"errors"
	"log/slog"
	"net/http"
	"reservoir/utils/syncmap"
	"time"
)

var (
	ErrNoSession = errors.New("no user session found")
)

type Session struct {
	ID                   string
	UserID               int64
	CreatedAt, ExpiresAt time.Time
}

const defaultLifetime = 1 * time.Hour
const extendThreshold = 10 * time.Minute
const gcInterval = 15 * time.Minute

var sessionStore *syncmap.SyncMap[string, *Session] = syncmap.New[string, *Session]()
var gcRunning = false

func StartSessionGC() {
	if gcRunning {
		return
	}

	slog.Info("Starting session garbage collector", "interval", gcInterval.String(), "session_lifetime", defaultLifetime.String())
	ticker := time.NewTicker(gcInterval)
	go func() {
		for range ticker.C {
			now := time.Now()
			for item := range sessionStore.Items() {
				if item.ExpiresAt.Before(now) {
					sessionStore.Delete(item.ID)
					slog.Debug("Deleted expired session", "session_id", item.ID)
				}
			}
		}
	}()
	gcRunning = true
}

func GetSession(sid string) (*Session, bool) {
	sess, ok := sessionStore.Get(sid)

	if !ok {
		return nil, false
	}

	// Extend session expiration if close to expiring
	if time.Until(sess.ExpiresAt) <= extendThreshold {
		slog.Debug("Session close to expiring, extending expiration", "session_id", sid, "expires_at", sess.ExpiresAt)
		sess.ExpiresAt = time.Now().Add(defaultLifetime)
		sessionStore.Set(sid, sess)
	}

	return sess, ok
}

func CreateSession(userId int64) *Session {
	slog.Debug("Creating new user session...", "user_id", userId)
	sid := rand.Text()
	now := time.Now()
	sess := &Session{
		ID:        sid,
		UserID:    userId,
		CreatedAt: now,
		ExpiresAt: now.Add(defaultLifetime),
	}

	sessionStore.Set(sid, sess)
	slog.Debug("Created new session", "session_id", sid, "expires_at", sess.ExpiresAt)

	return sess
}

func SessionFromRequest(r *http.Request) (sess *Session, ok bool) {
	sid, err := r.Cookie("reservoir.sid")
	if errors.Is(err, http.ErrNoCookie) {
		return nil, false
	}

	slog.Debug("Getting session from cookie...", "session_id", sid)
	sess, ok = GetSession(sid.Value)
	if !ok {
		return nil, false
	}
	slog.Debug("Got session from cookie", "session_id", sid, "expires_at", sess.ExpiresAt)

	return sess, true
}

func (s *Session) BuildSessionCookie() *http.Cookie {
	return &http.Cookie{
		Path:     "/",
		Name:     "reservoir.sid",
		Value:    s.ID,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
}

func (s *Session) Destroy() {
	sessionStore.Delete(s.ID)
	slog.Debug("Destroyed session", "session_id", s.ID)
}

// This package exists to avoid circular imports between api and the endpoints.
package apitypes

import (
	"net/http"
	"reservoir/cache"
	"reservoir/config"
	"reservoir/db/models"
	"reservoir/db/stores"
	"reservoir/webserver/auth"
)

type CacheController interface {
	CacheStats() cache.Stats
	ClearCache() error
}

type Context struct {
	Session        *auth.Session
	SessionManager *auth.SessionManager
	UserStore      *stores.UserStore
	Config         *config.Config
	Cache          CacheController
}

func CreateContext(r *http.Request, cfg *config.Config, sessions *auth.SessionManager, users *stores.UserStore, cacheController CacheController) (Context, error) {
	if sessions == nil {
		sessions = auth.DefaultSessionManager()
	}

	sess, _ := sessions.SessionFromRequest(r)

	return Context{
		Session:        sess,
		SessionManager: sessions,
		UserStore:      users,
		Config:         cfg,
		Cache:          cacheController,
	}, nil
}

func (c *Context) IsAuthenticated() bool {
	return c.Session != nil
}

func (c *Context) GetCurrentUser() (*models.User, error) {
	if !c.IsAuthenticated() {
		return nil, auth.ErrNoSession
	}
	if c.UserStore == nil {
		return nil, auth.ErrNoSession
	}
	return c.UserStore.GetByID(c.Session.UserID)
}

// This package exists to avoid circular imports between api and the endpoints.
package apitypes

import (
	"net/http"
	"reservoir/config"
	"reservoir/db/models"
	"reservoir/db/stores"
	"reservoir/webserver/auth"
)

type Context struct {
	Session        *auth.Session
	SessionManager *auth.SessionManager
	UserStore      *stores.UserStore
	Config         *config.Config
}

func CreateContext(r *http.Request, cfg *config.Config, sessions *auth.SessionManager) (Context, error) {
	if sessions == nil {
		sessions = auth.DefaultSessionManager()
	}

	sess, authorized := sessions.SessionFromRequest(r)
	var users *stores.UserStore
	if authorized {
		var err error
		users, err = stores.OpenUserStore()
		if err != nil {
			return Context{}, err
		}
	}

	return Context{
		Session:        sess,
		SessionManager: sessions,
		UserStore:      users,
		Config:         cfg,
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

func (c *Context) Close() {
	if c.UserStore != nil {
		c.UserStore.Close()
	}
}

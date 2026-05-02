// This package exists to avoid circular imports between api and the endpoints.
package apitypes

import (
	"net/http"
	"reservoir/cache"
	"reservoir/config"
	"reservoir/db/models"
	"reservoir/db/stores"
	"reservoir/utils/phc"
	"reservoir/webserver/auth"
)

type CacheController interface {
	CacheStats() cache.Stats
	ClearCache() error
}

type UserStore interface {
	Create(user *models.User) (*models.User, error)
	List() ([]models.User, error)
	GetByID(id int64) (*models.User, error)
	GetByUsername(username string) (*models.User, error)
	UpdateUsername(id int64, username string) (*models.User, error)
	UpdateAdmin(id int64, isAdmin bool) (*models.User, error)
	UpdatePassword(id int64, passwordHash phc.PHC, passwordChangeRequired bool) (*models.User, error)
	Delete(id int64) error
	Save(user *models.User) error
	Close() error
}

type Context struct {
	Session        *auth.Session
	SessionManager *auth.SessionManager
	UserStore      UserStore
	Config         *config.Config
	Cache          CacheController
}

func CreateContext(r *http.Request, cfg *config.Config, sessions *auth.SessionManager, cacheController CacheController) (Context, error) {
	if sessions == nil {
		sessions = auth.DefaultSessionManager()
	}

	sess, authorized := sessions.SessionFromRequest(r)
	var users UserStore
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

func (c *Context) Close() {
	if c.UserStore != nil {
		c.UserStore.Close()
	}
}

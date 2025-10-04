// This package exists to avoid circular imports between api and the endpoints.
package apitypes

import (
	"net/http"
	"reservoir/db/models"
	"reservoir/db/stores"
	"reservoir/webserver/auth"
)

type Context struct {
	Session   *auth.Session
	UserStore *stores.UserStore
}

func CreateContext(r *http.Request) (Context, error) {
	sess, authorized := auth.SessionFromRequest(r)
	var users *stores.UserStore
	if authorized {
		users, _ = stores.OpenUserStore()
	}

	return Context{
		Session:   sess,
		UserStore: users,
	}, nil
}

func (c *Context) IsAuthenticated() bool {
	return c.Session != nil
}

func (c *Context) GetCurrentUser() (*models.User, error) {
	if !c.IsAuthenticated() {
		return nil, auth.ErrNoSession
	}
	return c.UserStore.GetByID(c.Session.UserID)
}

func (c *Context) Close() {
	if c.UserStore != nil {
		c.UserStore.Close()
	}
}

// This package exists to avoid circular imports between api and the endpoints.
package apitypes

import (
	"net/http"
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

func (c *Context) Close() {
	if c.UserStore != nil {
		c.UserStore.Close()
	}
}

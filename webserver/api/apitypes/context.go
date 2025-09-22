// This package exists to avoid circular imports between api and the endpoints.
package apitypes

import (
	"net/http"
	"reservoir/webserver/auth"
)

type Context struct {
	Session *auth.Session
}

func CreateContext(r *http.Request) Context {
	sess, _ := auth.SessionFromRequest(r)
	return Context{
		Session: sess,
	}
}

func WrapWithContext(methodFunc MethodFunc, ensureAuthenticated bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := CreateContext(r)
		if ensureAuthenticated && !ctx.IsAuthenticated() {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		methodFunc(w, r, ctx)
	}
}

func (c *Context) IsAuthenticated() bool {
	return c.Session != nil
}

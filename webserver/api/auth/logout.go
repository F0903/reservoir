package auth

import (
	"net/http"
	"reservoir/webserver/api/apitypes"
	"time"
)

type LogoutEndpoint struct{}

func (e *LogoutEndpoint) Path() string {
	return "/auth/logout"
}

func (e *LogoutEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method: "POST",
			Func:   e.Post,
		},
	}
}

func (e *LogoutEndpoint) Post(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	if !ctx.IsAuthenticated() {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ctx.Session.Destroy()

	cookie := ctx.Session.BuildSessionCookie()
	cookie.Value = ""
	cookie.Expires = time.Unix(0, 0).UTC()
	cookie.MaxAge = -1

	http.SetCookie(w, cookie)
	w.Write([]byte("Logged out successfully"))
}

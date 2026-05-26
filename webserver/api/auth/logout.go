package auth

import (
	"log/slog"
	"net/http"
	appservices "reservoir/application/services"
	"reservoir/webserver/api/apihttp"
	"reservoir/webserver/api/apitypes"
)

type LogoutEndpoint struct{}

func (e *LogoutEndpoint) Path() string {
	return "/auth/logout"
}

func (e *LogoutEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:                      "POST",
			Func:                        e.Post,
			RequiresAuth:                true,
			AllowPasswordChangeRequired: true,
		},
	}
}

func (e *LogoutEndpoint) Post(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	cookie, err := ctx.Services.Auth.Logout(ctx.Request)
	if err != nil {
		if err == appservices.ErrNoSession {
			apihttp.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		slog.Error("Error during logout", "error", err)
		apihttp.InternalServerError(w)
		return
	}

	http.SetCookie(w, cookie)
	apihttp.NoContent(w)
}

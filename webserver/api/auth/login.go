package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"reservoir/webserver/api/apihttp"
	"reservoir/webserver/api/apitypes"
	"reservoir/webserver/api/auth/models"
	"reservoir/webserver/auth"
)

type LoginEndpoint struct{}

func (e *LoginEndpoint) Path() string {
	return "/auth/login"
}

func (e *LoginEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:       "POST",
			Func:         e.Post,
			RequiresAuth: false,
		},
	}
}

func (e *LoginEndpoint) Post(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	if !apihttp.RequireJSONContentType(w, r) {
		return
	}

	var creds auth.Credentials
	if !apihttp.DecodeJSON(w, r, &creds) {
		return
	}

	if ctx.IsAuthenticated() {
		apihttp.WriteText(w, http.StatusOK, "Already Authenticated")
		return
	}

	user, err := creds.Authenticate()
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			apihttp.Error(w, "Invalid Credentials", http.StatusUnauthorized)
			return
		}
		slog.Error("Error during authentication", "error", err)
		apihttp.InternalServerError(w)
		return
	}

	sess := ctx.SessionManager.Create(user.ID)
	http.SetCookie(w, sess.BuildSessionCookie())

	apihttp.WriteJSON(w, http.StatusOK, models.FromUser(user))
}

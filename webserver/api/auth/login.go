package auth

import (
	"errors"
	"log/slog"
	"net/http"
	appservices "reservoir/application/services"
	"reservoir/webserver/api/apihttp"
	"reservoir/webserver/api/apitypes"
	"reservoir/webserver/api/auth/models"
)

type LoginEndpoint struct{}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

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

	var req loginRequest
	if !apihttp.DecodeJSON(w, r, &req) {
		return
	}

	user, cookie, err := ctx.Services.Auth.Login(ctx.Request, req.Username, req.Password)
	if err != nil {
		if errors.Is(err, appservices.ErrAlreadyAuthenticated) {
			apihttp.WriteText(w, http.StatusOK, "Already Authenticated")
			return
		}
		if errors.Is(err, appservices.ErrInvalidCredentials) {
			apihttp.Error(w, "Invalid Credentials", http.StatusUnauthorized)
			return
		}
		slog.Error("Error during authentication", "error", err)
		apihttp.InternalServerError(w)
		return
	}

	http.SetCookie(w, cookie)

	apihttp.WriteJSON(w, http.StatusOK, models.FromUser(user))
}

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

type MeEndpoint struct{}

type updateMeRequest struct {
	Username string `json:"username"`
}

func (e *MeEndpoint) Path() string {
	return "/auth/me"
}

func (e *MeEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:                      "GET",
			Func:                        e.Get,
			RequiresAuth:                true,
			AllowPasswordChangeRequired: true,
		},
		{
			Method:       "PATCH",
			Func:         e.Patch,
			RequiresAuth: true,
		},
	}
}

func (e *MeEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	user, err := ctx.GetCurrentUser()
	if err != nil {
		slog.Error("Error retrieving current user", "error", err)
		apihttp.InternalServerError(w)
		return
	}

	apihttp.WriteJSON(w, http.StatusOK, models.FromUser(user))
}

func (e *MeEndpoint) Patch(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	if !apihttp.RequireJSONContentType(w, r) {
		return
	}

	var req updateMeRequest
	if !apihttp.DecodeJSON(w, r, &req) {
		return
	}

	updated, err := ctx.Services.Auth.UpdateCurrentUsername(ctx.Request, req.Username)
	if err != nil {
		switch {
		case errors.Is(err, appservices.ErrUsernameEmpty):
			apihttp.BadRequest(w, "Username must not be empty")
		case errors.Is(err, appservices.ErrUsernameTaken):
			apihttp.Error(w, "Username is already taken", http.StatusConflict)
		case errors.Is(err, appservices.ErrUserNotFound):
			apihttp.Error(w, "User not found", http.StatusNotFound)
		default:
			slog.Error("Error updating current user", "error", err)
			apihttp.InternalServerError(w)
		}
		return
	}

	apihttp.WriteJSON(w, http.StatusOK, models.FromUser(updated))
}

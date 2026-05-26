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

type BootstrapEndpoint struct{}

type bootstrapStatusResponse struct {
	BootstrapRequired bool `json:"bootstrap_required"`
}

type bootstrapRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (e *BootstrapEndpoint) Path() string {
	return "/auth/bootstrap"
}

func (e *BootstrapEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:       http.MethodGet,
			Func:         e.Get,
			RequiresAuth: false,
		},
		{
			Method:       http.MethodPost,
			Func:         e.Post,
			RequiresAuth: false,
		},
	}
}

func (e *BootstrapEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	required, err := ctx.Services.Auth.BootstrapRequired()
	if err != nil {
		slog.Error("Error checking bootstrap status", "error", err)
		apihttp.InternalServerError(w)
		return
	}

	apihttp.WriteJSON(w, http.StatusOK, bootstrapStatusResponse{BootstrapRequired: required})
}

func (e *BootstrapEndpoint) Post(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	if !apihttp.RequireJSONContentType(w, r) {
		return
	}

	var req bootstrapRequest
	if !apihttp.DecodeJSON(w, r, &req) {
		return
	}

	user, cookie, err := ctx.Services.Auth.CreateBootstrapAdmin(req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, appservices.ErrBootstrapNotRequired):
			apihttp.Error(w, "Bootstrap is not required", http.StatusConflict)
		case errors.Is(err, appservices.ErrUsernameEmpty):
			apihttp.BadRequest(w, "Username must not be empty")
		case errors.Is(err, appservices.ErrPasswordEmpty):
			apihttp.BadRequest(w, "Password must not be empty")
		case errors.Is(err, appservices.ErrPasswordTooShort):
			apihttp.BadRequest(w, "Password must be at least 12 characters")
		default:
			slog.Error("Error creating bootstrap admin", "error", err)
			apihttp.InternalServerError(w)
		}
		return
	}

	http.SetCookie(w, cookie)

	apihttp.WriteJSON(w, http.StatusCreated, models.FromUser(user))
}

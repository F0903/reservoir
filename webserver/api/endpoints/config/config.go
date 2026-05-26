package config

import (
	"errors"
	"log/slog"
	"net/http"
	appservices "reservoir/application/services"
	"reservoir/webserver/api/apihttp"
	"reservoir/webserver/api/apitypes"
)

const successResponse = "success"
const restartRequiredResponse = "restart required"

type ConfigEndpoint struct{}

func (e *ConfigEndpoint) Path() string {
	return "/config"
}

func (e *ConfigEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:        "GET",
			Func:          e.Get,
			RequiresAuth:  true,
			RequiresAdmin: true,
		},
		{
			Method:        "PATCH",
			Func:          e.Patch,
			RequiresAuth:  true,
			RequiresAdmin: true,
		},
	}
}

func (e *ConfigEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	cfg, err := ctx.Services.Config.Snapshot()
	if err != nil {
		slog.Error("Failed to get config", "error", err)
		apihttp.InternalServerError(w)
		return
	}

	apihttp.WriteJSON(w, http.StatusOK, cfg)
}

func (e *ConfigEndpoint) Patch(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	if !apihttp.RequireJSONContentType(w, r) {
		return
	}

	var updates map[string]any
	if !apihttp.DecodeJSON(w, r, &updates) {
		return
	}

	status, err := ctx.Services.Config.UpdatePartial(updates)
	if err != nil {
		if errors.Is(err, appservices.ErrConfigValidationFailed) {
			apihttp.BadRequest(w, err.Error())
			return
		}

		slog.Error("Failed to partially update config", "error", err)
		apihttp.InternalServerError(w)
		return
	}

	switch status {
	case appservices.ConfigUpdateSuccess:
		apihttp.WriteText(w, http.StatusAccepted, successResponse)
	case appservices.ConfigUpdateRestartRequired:
		apihttp.WriteText(w, http.StatusAccepted, restartRequiredResponse)
	}
}

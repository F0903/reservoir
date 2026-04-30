package config

import (
	"log/slog"
	"net/http"
	"reservoir/config"
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
			Method:       "GET",
			Func:         e.Get,
			RequiresAuth: true,
		},
		{
			Method:       "PATCH",
			Func:         e.Patch,
			RequiresAuth: true,
		},
	}
}

func (e *ConfigEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	apihttp.WriteJSON(w, http.StatusOK, ctx.Config)
}

func (e *ConfigEndpoint) Patch(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	if !apihttp.RequireJSONContentType(w, r) {
		return
	}

	var updates map[string]any
	if !apihttp.DecodeJSON(w, r, &updates) {
		return
	}

	status, err := config.UpdatePartialFromConfig(ctx.Config, updates)
	if err != nil {
		slog.Error("Failed to partially update config", "error", err)
		apihttp.InternalServerError(w)
		return
	}

	switch status {
	case config.UpdateStatusSuccess:
		apihttp.WriteText(w, http.StatusAccepted, successResponse)
	case config.UpdateStatusRestartRequired:
		apihttp.WriteText(w, http.StatusAccepted, restartRequiredResponse)
	}
}

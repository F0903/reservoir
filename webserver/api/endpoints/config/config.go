package config

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reservoir/config"
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
	cfgLock := config.Global.Immutable()
	cfg := cfgLock.Copy()

	responseJson, err := json.Marshal(cfg)
	if err != nil {
		slog.Error("Error marshaling config to JSON", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}

func (e *ConfigEndpoint) Patch(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	// Parse JSON into a map
	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		slog.Error("Error decoding JSON", "error", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	status, err := config.UpdatePartialFromJSON(updates)
	if err != nil {
		slog.Error("Failed to partially update config", "error", err)
		http.Error(w, "Failed to update config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	switch status {
	case config.UpdateStatusSuccess:
		w.Write([]byte(successResponse))
	case config.UpdateStatusRestartRequired:
		w.Write([]byte(restartRequiredResponse))
	}
}

package config

import (
	"apt_cacher_go/config"
	"apt_cacher_go/webserver/api/apitypes"
	"encoding/json"
	"log/slog"
	"net/http"
)

type configUpdateResponse struct {
	Success       bool   `json:"success"`
	RestartNeeded bool   `json:"restart_needed"`
	Message       string `json:"message"`
}

type ConfigEndpoint struct{}

func (m *ConfigEndpoint) Path() string {
	return "/settings"
}

func (m *ConfigEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method: "GET",
			Func:   m.Get,
		},
		{
			Method: "POST",
			Func:   m.Post,
		},
	}
}

func (m *ConfigEndpoint) Get(w http.ResponseWriter, r *http.Request) {
	configJson, err := json.Marshal(config.Global)
	if err != nil {
		slog.Error("Error marshaling config", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(configJson)
}

func (m *ConfigEndpoint) Post(w http.ResponseWriter, r *http.Request) {
	var newConfig config.Config
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		slog.Error("Error decoding config", "error", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	config.Global = &newConfig

	response := configUpdateResponse{
		Success:       true,
		RestartNeeded: true,
		Message:       "Configuration updated successfully",
	}

	responseJson, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}

//TODO: Patch endpoint to partially update config

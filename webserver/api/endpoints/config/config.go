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
	return "/config"
}

func (m *ConfigEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method: "GET",
			Func:   m.Get,
		},
		{
			Method: "PUT",
			Func:   m.Put,
		},
		{
			Method: "PATCH",
			Func:   m.Patch,
		},
	}
}

func (m *ConfigEndpoint) Get(w http.ResponseWriter, r *http.Request) {
	config := config.Get()
	configJson, err := json.Marshal(config)
	if err != nil {
		slog.Error("Error marshaling config", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(configJson)
}

func (m *ConfigEndpoint) Put(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	var newConfig config.Config
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		slog.Error("Error decoding config", "error", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := config.Update(func(cfg *config.Config) {
		*cfg = newConfig
	}); err != nil {
		slog.Error("Error updating config", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := configUpdateResponse{
		Success:       true,
		RestartNeeded: false,
		Message:       "Configuration updated successfully",
	}

	responseJson, err := json.Marshal(response)
	if err != nil {
		slog.Error("Error marshaling response", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}

func (m *ConfigEndpoint) Patch(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	http.Error(w, "PATCH method is not implemented", http.StatusNotImplemented)
}

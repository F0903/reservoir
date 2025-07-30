package config

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reservoir/config"
	"reservoir/webserver/api/apitypes"
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
			Method: "PATCH",
			Func:   m.Patch,
		},
	}
}

func (m *ConfigEndpoint) Patch(w http.ResponseWriter, r *http.Request) {
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

	if err := config.UpdatePartialFromJSON(updates); err != nil {
		slog.Error("Failed to partially update config", "error", err)
		http.Error(w, "Failed to update config", http.StatusInternalServerError)
		return
	}

	responseJson, err := json.Marshal(configUpdateResponse{
		Success:       true,
		RestartNeeded: false,
		Message:       "Configuration updated successfully",
	})
	if err != nil {
		slog.Error("Error marshaling response", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}

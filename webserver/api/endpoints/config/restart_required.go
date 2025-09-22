package config

import (
	"encoding/json"
	"net/http"
	"reservoir/config"
	"reservoir/webserver/api/apitypes"
)

type RestartRequiredEndpoint struct{}

func (e *RestartRequiredEndpoint) Path() string {
	return "/config/restart-required"
}

func (e *RestartRequiredEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method: "GET",
			Func:   e.Get,
		},
	}
}

func (e *RestartRequiredEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	restart := config.IsRestartNeeded()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"restart_required": restart})
}

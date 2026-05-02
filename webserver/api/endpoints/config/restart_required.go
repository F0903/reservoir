package config

import (
	"net/http"
	"reservoir/config"
	"reservoir/webserver/api/apihttp"
	"reservoir/webserver/api/apitypes"
)

type RestartRequiredEndpoint struct{}

func (e *RestartRequiredEndpoint) Path() string {
	return "/config/restart-required"
}

func (e *RestartRequiredEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:        "GET",
			Func:          e.Get,
			RequiresAuth:  true,
			RequiresAdmin: true,
		},
	}
}

func (e *RestartRequiredEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	restart := config.IsRestartNeeded()
	apihttp.WriteJSON(w, http.StatusOK, map[string]bool{"restart_required": restart})
}

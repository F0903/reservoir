package version

import (
	"encoding/json"
	"net/http"
	"reservoir/version"
	"reservoir/webserver/api/apitypes"
)

type VersionEndpoint struct{}

func (e *VersionEndpoint) Path() string {
	return "/version"
}

func (e *VersionEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:       "GET",
			Func:         e.Get,
			RequiresAuth: true,
		},
	}
}

func (e *VersionEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"version": version.Version})
}

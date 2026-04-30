package version

import (
	"net/http"
	"reservoir/version"
	"reservoir/webserver/api/apihttp"
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
	apihttp.WriteJSON(w, http.StatusOK, map[string]string{"version": version.Version})
}

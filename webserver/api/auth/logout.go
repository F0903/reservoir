package auth

import (
	"net/http"
	"reservoir/webserver/api/apitypes"
)

type LogoutEndpoint struct{}

func (e *LogoutEndpoint) Path() string {
	return "/auth/logout"
}

func (e *LogoutEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method: "POST",
			Func:   e.Post,
		},
	}
}

func (e *LogoutEndpoint) Post(w http.ResponseWriter, r *http.Request, ctx *apitypes.Context) {
	//TODO: Handle logout authentication logic here
}

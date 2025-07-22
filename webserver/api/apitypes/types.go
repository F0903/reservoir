package apitypes

import "net/http"

type EndpointMethod struct {
	Method string
	Func   http.HandlerFunc
}

type Endpoint interface {
	Path() string
	EndpointMethods() []EndpointMethod
}

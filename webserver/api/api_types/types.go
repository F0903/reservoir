package api_types

import "net/http"

type EndpointMethod struct {
	Method string
	Func   http.HandlerFunc
}

type Endpoint interface {
	Path() string
	EndpointMethods() []EndpointMethod
}

// This package exists to avoid circular imports between api and the endpoints.
package apitypes

import (
	"net/http"
)

type MethodFunc func(w http.ResponseWriter, r *http.Request, ctx Context)

type EndpointMethod struct {
	Method       string
	Func         MethodFunc
	RequiresAuth bool
}

type Endpoint interface {
	Path() string
	EndpointMethods() []EndpointMethod
}

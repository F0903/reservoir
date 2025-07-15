package api

import "net/http"

type apiEndpoint interface {
	Path() string
	Method() string
	Endpoint(w http.ResponseWriter, r *http.Request)
}

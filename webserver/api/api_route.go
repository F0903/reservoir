package api

import "net/http"

type apiRouteHandler = func(uri string, w http.ResponseWriter, r *http.Request) error

type apiRoute interface {
	Name() string
	HandleRoute(path string, w http.ResponseWriter, r *http.Request) error
}

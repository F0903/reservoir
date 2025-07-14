package api

import (
	"apt_cacher_go/webserver/api/routes"
	"net/http"
	"strings"
)

type apiHandler struct {
	routes map[string]apiRouteHandler
}

func newAPIHandler() *apiHandler {
	h := &apiHandler{
		routes: make(map[string]apiRouteHandler),
	}
	h.registerDefaultRoutes()
	return h
}

func (h *apiHandler) registerDefaultRoutes() error {
	// Register all our current API routes here.

	h.RegisterRoute(&routes.MetricsRoute{})

	return nil
}

func (h *apiHandler) RegisterRoute(route apiRoute) {
	// We "cache" the name and function in a map for quick lookup later
	h.routes[route.Name()] = route.HandleRoute
}

func (h *apiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	remainingPath, ok := strings.CutPrefix(path, "/api/")
	if !ok {
		// If the path doesn't start with "/api/", return 404
		http.NotFound(w, r)
		return
	}

	endpoint, remainder := parseNextPathComponent(remainingPath)
	if endpoint == "" {
		// If the endpoint is empty (meaning the URI was just "/api/"), return 404
		http.NotFound(w, r)
		return
	}

	if route, ok := h.routes[endpoint]; ok {
		route(remainder, w, r)
	} else {
		http.NotFound(w, r)
	}
}

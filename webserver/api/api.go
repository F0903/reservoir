package api

import (
	"apt_cacher_go/webserver/api/endpoints/metrics"
	"fmt"
	"net/http"
)

type API struct {
	basePath  string
	endpoints []apiEndpoint
}

func New() *API {
	return &API{
		basePath: "/api",
		endpoints: []apiEndpoint{
			// Register all our current API routes here.
			&metrics.GetAllMetricsEndpoint{},
			&metrics.GetCacheMetricsEndpoint{},
			&metrics.GetRequestsMetricsEndpoint{},
			&metrics.GetTimingMetricsEndpoint{},
		},
	}
}

func (api *API) RegisterHandlers(mux *http.ServeMux) error {

	for _, endpoint := range api.endpoints {
		pattern := fmt.Sprintf("%s %s%s", endpoint.Method(), api.basePath, endpoint.Path())
		mux.HandleFunc(pattern, endpoint.Endpoint)
	}

	return nil
}

package api

import (
	"apt_cacher_go/webserver/api/api_types"
	"apt_cacher_go/webserver/api/endpoints/metrics"
	"fmt"
	"net/http"
)

type API struct {
	basePath  string
	endpoints []api_types.Endpoint
}

func New() *API {
	return &API{
		basePath: "/api",
		endpoints: []api_types.Endpoint{
			// Register all our current API routes here.
			&metrics.AllMetricsEndpoint{},
			&metrics.CacheMetricsEndpoint{},
			&metrics.RequestsMetricsEndpoint{},
			&metrics.TimingMetricsEndpoint{},
		},
	}
}

func (api *API) RegisterHandlers(mux *http.ServeMux) error {

	for _, endpoint := range api.endpoints {
		for _, method := range endpoint.EndpointMethods() {
			if method.Method == "" {
				return fmt.Errorf("endpoint %s has no method defined", endpoint.Path())
			}

			if method.Func == nil {
				return fmt.Errorf("endpoint %s has no function defined for method %s", endpoint.Path(), method.Method)
			}

			pattern := fmt.Sprintf("%s %s%s", method.Method, api.basePath, endpoint.Path())
			mux.HandleFunc(pattern, method.Func)
		}
	}

	return nil
}

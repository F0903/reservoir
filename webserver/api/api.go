package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reservoir/webserver/api/apitypes"
	"reservoir/webserver/api/endpoints/config"
	"reservoir/webserver/api/endpoints/log"
	"reservoir/webserver/api/endpoints/metrics"
)

var (
	ErrEndpointNoMethod   = errors.New("endpoint has no method defined")
	ErrEndpointNoFunction = errors.New("endpoint has no function defined")
)

type API struct {
	basePath  string
	endpoints []apitypes.Endpoint
}

func New() *API {
	return &API{
		basePath: "/api",
		endpoints: []apitypes.Endpoint{
			// Register all our current API endpoints here.
			&metrics.AllMetricsEndpoint{},
			&metrics.CacheMetricsEndpoint{},
			&metrics.RequestsMetricsEndpoint{},
			&metrics.SystemMetricsEndpoint{},
			&config.ConfigEndpoint{},
			&config.RestartRequiredEndpoint{},
			&log.LogEndpoint{},
			&log.LogStreamEndpoint{},
		},
	}
}

func (api *API) RegisterHandlers(mux *http.ServeMux) error {
	for _, endpoint := range api.endpoints {
		for _, method := range endpoint.EndpointMethods() {
			if method.Method == "" {
				slog.Error("Endpoint has no method defined", "endpoint_path", endpoint.Path())
				return fmt.Errorf("%w: endpoint '%s'", ErrEndpointNoMethod, endpoint.Path())
			}

			if method.Func == nil {
				slog.Error("Endpoint has no function defined", "endpoint_path", endpoint.Path(), "method", method.Method)
				return fmt.Errorf("%w: endpoint '%s' method '%s'", ErrEndpointNoFunction, endpoint.Path(), method.Method)
			}

			pattern := fmt.Sprintf("%s %s%s", method.Method, api.basePath, endpoint.Path())
			mux.HandleFunc(pattern, apitypes.WrapWithContext(method.Func, true))
			slog.Debug("Registered API handler", "pattern", pattern, "endpoint", endpoint.Path())
		}
	}

	slog.Info("Successfully registered all API handlers", "endpoint_count", len(api.endpoints))
	return nil
}

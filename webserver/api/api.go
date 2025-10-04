package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reservoir/webserver/api/apitypes"
	"reservoir/webserver/api/auth"
	"reservoir/webserver/api/endpoints/config"
	"reservoir/webserver/api/endpoints/log"
	"reservoir/webserver/api/endpoints/metrics"
)

var (
	ErrEndpointNoMethod     = errors.New("endpoint has no method defined")
	ErrEndpointNoFunction   = errors.New("endpoint has no function defined")
	ErrEndpointUnauthorized = errors.New("authentication required")
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
			&auth.LoginEndpoint{},
			&auth.LogoutEndpoint{},
			&auth.ChangePasswordEndpoint{},
		},
	}
}

func EnsureAllowed(ctx apitypes.Context, method apitypes.EndpointMethod) (statusCode int, err error) {
	if !method.RequiresAuth {
		return http.StatusOK, nil
	}

	if !ctx.IsAuthenticated() {
		return http.StatusUnauthorized, ErrEndpointUnauthorized
	}

	return http.StatusOK, nil
}

func WrapHandler(methodFunc apitypes.MethodFunc, preRunHook func(apitypes.Context) (statusCode int, err error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := apitypes.CreateContext(r)
		if err != nil {
			slog.Error("Error creating request context", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer ctx.Close()

		if preRunHook != nil {
			if status, err := preRunHook(ctx); err != nil {
				if err == ErrEndpointUnauthorized {
					http.Error(w, "Unauthorized", status)
					return
				}

				slog.Warn("Pre-run hook failed", "error", err, "status", status)
				http.Error(w, "Internal Server Error", status)
				return
			}
		}

		methodFunc(w, r, ctx)
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
			mux.HandleFunc(pattern, WrapHandler(method.Func, func(ctx apitypes.Context) (int, error) {
				return EnsureAllowed(ctx, method)
			}))
			slog.Debug("Registered API handler", "pattern", pattern, "endpoint", endpoint.Path())
		}
	}

	slog.Info("Successfully registered all API handlers", "endpoint_count", len(api.endpoints))
	return nil
}

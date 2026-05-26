package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	appservices "reservoir/application/services"
	dbmodels "reservoir/db/models"
	"reservoir/webserver/api/apihttp"
	"reservoir/webserver/api/apitypes"
	authEndpoints "reservoir/webserver/api/auth"
	cacheEndpoint "reservoir/webserver/api/endpoints/cache"
	configEndpoint "reservoir/webserver/api/endpoints/config"
	"reservoir/webserver/api/endpoints/log"
	"reservoir/webserver/api/endpoints/metrics"
	"reservoir/webserver/api/endpoints/version"
)

var (
	ErrEndpointNoMethod      = errors.New("endpoint has no method defined")
	ErrEndpointNoFunction    = errors.New("endpoint has no function defined")
	ErrEndpointUnauthorized  = errors.New("authentication required")
	ErrEndpointAdminRequired = errors.New("administrator access required")
	ErrPasswordChangeNeeded  = errors.New("password change required")
)

type API struct {
	basePath  string
	services  *appservices.Container
	endpoints []apitypes.Endpoint
}

func New(services *appservices.Container) *API {
	return &API{
		basePath: "/api",
		services: services,
		endpoints: []apitypes.Endpoint{
			// Register all our current API endpoints here.
			&version.VersionEndpoint{},
			&cacheEndpoint.StatusEndpoint{},
			&cacheEndpoint.ClearEndpoint{},
			&metrics.AllMetricsEndpoint{},
			&metrics.CacheMetricsEndpoint{},
			&metrics.RequestsMetricsEndpoint{},
			&metrics.SystemMetricsEndpoint{},
			&configEndpoint.ConfigEndpoint{},
			&configEndpoint.RestartRequiredEndpoint{},
			&log.LogEndpoint{},
			&log.LogStreamEndpoint{},
			&authEndpoints.BootstrapEndpoint{},
			&authEndpoints.LoginEndpoint{},
			&authEndpoints.LogoutEndpoint{},
			&authEndpoints.MeEndpoint{},
			&authEndpoints.ChangePasswordEndpoint{},
			&authEndpoints.UsersEndpoint{},
			&authEndpoints.UserEndpoint{},
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

	user, err := ctx.GetCurrentUser()
	if err != nil {
		if errors.Is(err, appservices.ErrUserNotFound) {
			return http.StatusUnauthorized, ErrEndpointUnauthorized
		}
		return http.StatusInternalServerError, err
	}
	if user == nil {
		return http.StatusUnauthorized, ErrEndpointUnauthorized
	}
	if !adminAllowed(user, method) {
		return http.StatusForbidden, ErrEndpointAdminRequired
	}
	if !passwordChangeAllowed(user, method) {
		return http.StatusForbidden, ErrPasswordChangeNeeded
	}

	return http.StatusOK, nil
}

func passwordChangeAllowed(user *dbmodels.User, method apitypes.EndpointMethod) bool {
	return !user.PasswordChangeRequired || method.AllowPasswordChangeRequired
}

func adminAllowed(user *dbmodels.User, method apitypes.EndpointMethod) bool {
	return !method.RequiresAdmin || user.IsAdmin
}

func WrapHandler(services *appservices.Container, methodFunc apitypes.MethodFunc, preRunHook func(apitypes.Context) (statusCode int, err error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := apitypes.CreateContext(r, services)
		if err != nil {
			slog.Error("Error creating request context", "error", err)
			apihttp.InternalServerError(w)
			return
		}

		if preRunHook != nil {
			if status, err := preRunHook(ctx); err != nil {
				if err == ErrEndpointUnauthorized {
					apihttp.Error(w, "Unauthorized", status)
					return
				}
				if err == ErrPasswordChangeNeeded {
					apihttp.Error(w, "Password change required", status)
					return
				}
				if err == ErrEndpointAdminRequired {
					apihttp.Error(w, "Administrator access required", status)
					return
				}

				slog.Warn("Pre-run hook failed", "error", err, "status", status)
				apihttp.Error(w, "Internal Server Error", status)
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
			mux.HandleFunc(pattern, WrapHandler(api.services, method.Func, func(ctx apitypes.Context) (int, error) {
				return EnsureAllowed(ctx, method)
			}))
			slog.Debug("Registered API handler", "pattern", pattern, "endpoint", endpoint.Path())
		}
	}

	slog.Info("Successfully registered all API handlers", "endpoint_count", len(api.endpoints))
	return nil
}

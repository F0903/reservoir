package metrics

import (
	"net/http"
	"reservoir/webserver/api/apihttp"
	"reservoir/webserver/api/apitypes"
)

type CacheMetricsEndpoint struct{}

func (m *CacheMetricsEndpoint) Path() string {
	return "/metrics/cache"
}

func (m *CacheMetricsEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:       "GET",
			Func:         m.Get,
			RequiresAuth: true,
		},
	}
}

func (m *CacheMetricsEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	runtimeMetrics, ok := metricsFromContext(ctx)
	if !ok {
		apihttp.InternalServerError(w)
		return
	}

	collectCacheStorage(ctx, runtimeMetrics)

	apihttp.WriteJSON(w, http.StatusOK, runtimeMetrics.Cache)
}

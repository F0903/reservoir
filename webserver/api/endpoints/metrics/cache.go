package metrics

import (
	"net/http"
	"reservoir/metrics"
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
			Method:        "GET",
			Func:          m.Get,
			RequiresAuth:  true,
			RequiresAdmin: true,
		},
	}
}

func (m *CacheMetricsEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	collectCacheStorage(ctx)

	apihttp.WriteJSON(w, http.StatusOK, metrics.Global.Cache)
}

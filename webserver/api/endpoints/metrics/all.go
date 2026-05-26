package metrics

import (
	"net/http"
	"reservoir/webserver/api/apihttp"
	"reservoir/webserver/api/apitypes"
)

type AllMetricsEndpoint struct{}

func (m *AllMetricsEndpoint) Path() string {
	return "/metrics"
}

func (m *AllMetricsEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:       "GET",
			Func:         m.Get,
			RequiresAuth: true,
		},
	}
}

func (m *AllMetricsEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	runtimeMetrics, ok := metricsFromContext(ctx)
	if !ok {
		apihttp.InternalServerError(w)
		return
	}

	runtimeMetrics.RunCollectors() // Run all collectors to make sure we get updated metrics
	collectCacheStorage(ctx, runtimeMetrics)

	apihttp.WriteJSON(w, http.StatusOK, runtimeMetrics)
}

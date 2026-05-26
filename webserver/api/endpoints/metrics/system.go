package metrics

import (
	"net/http"
	"reservoir/webserver/api/apihttp"
	"reservoir/webserver/api/apitypes"
)

type SystemMetricsEndpoint struct{}

func (m *SystemMetricsEndpoint) Path() string {
	return "/metrics/system"
}

func (m *SystemMetricsEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:       "GET",
			Func:         m.Get,
			RequiresAuth: true,
		},
	}
}

func (m *SystemMetricsEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	runtimeMetrics, ok := metricsFromContext(ctx)
	if !ok {
		apihttp.InternalServerError(w)
		return
	}

	runtimeMetrics.System.Collect() // Run the system metrics collector

	apihttp.WriteJSON(w, http.StatusOK, runtimeMetrics.System)
}

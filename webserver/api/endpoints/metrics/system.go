package metrics

import (
	"net/http"
	"reservoir/metrics"
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
			Method:        "GET",
			Func:          m.Get,
			RequiresAuth:  true,
			RequiresAdmin: true,
		},
	}
}

func (m *SystemMetricsEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	metrics.Global.System.Collect() // Run the system metrics collector

	apihttp.WriteJSON(w, http.StatusOK, metrics.Global.System)
}

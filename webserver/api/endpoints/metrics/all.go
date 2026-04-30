package metrics

import (
	"net/http"
	"reservoir/metrics"
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
	metrics.Global.RunCollectors() // Run all collectors to make sure we get updated metrics

	apihttp.WriteJSON(w, http.StatusOK, metrics.Global)
}

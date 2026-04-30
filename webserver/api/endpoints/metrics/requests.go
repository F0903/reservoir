package metrics

import (
	"net/http"
	"reservoir/metrics"
	"reservoir/webserver/api/apihttp"
	"reservoir/webserver/api/apitypes"
)

type RequestsMetricsEndpoint struct{}

func (m *RequestsMetricsEndpoint) Path() string {
	return "/metrics/requests"
}

func (m *RequestsMetricsEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:       "GET",
			Func:         m.Get,
			RequiresAuth: true,
		},
	}
}

func (m *RequestsMetricsEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	apihttp.WriteJSON(w, http.StatusOK, metrics.Global.Requests)
}

package metrics

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reservoir/metrics"
	"reservoir/webserver/api/apitypes"
)

type RequestsMetricsEndpoint struct{}

func (m *RequestsMetricsEndpoint) Path() string {
	return "/metrics/requests"
}

func (m *RequestsMetricsEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method: "GET",
			Func:   m.Get,
		},
	}
}

func (m *RequestsMetricsEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	requestsJson, err := json.Marshal(metrics.Global.Requests)
	if err != nil {
		slog.Error("Error marshaling requests metrics", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(requestsJson)
}

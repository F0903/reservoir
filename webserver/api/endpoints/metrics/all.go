package metrics

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reservoir/metrics"
	"reservoir/webserver/api/apitypes"
)

type AllMetricsEndpoint struct{}

func (m *AllMetricsEndpoint) Path() string {
	return "/metrics"
}

func (m *AllMetricsEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method: "GET",
			Func:   m.Get,
		},
	}
}

func (m *AllMetricsEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	metricsJson, err := json.Marshal(metrics.Global)
	if err != nil {
		slog.Error("Error marshaling all metrics", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(metricsJson)
}

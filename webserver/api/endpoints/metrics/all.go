package metrics

import (
	"apt_cacher_go/metrics"
	"apt_cacher_go/webserver/api/apitypes"
	"encoding/json"
	"log/slog"
	"net/http"
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

func (m *AllMetricsEndpoint) Get(w http.ResponseWriter, r *http.Request) {
	metricsJson, err := json.Marshal(metrics.Global)
	if err != nil {
		slog.Error("Error marshaling all metrics", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(metricsJson)
}

package metrics

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reservoir/metrics"
	"reservoir/webserver/api/apitypes"
)

type SystemMetricsEndpoint struct{}

func (m *SystemMetricsEndpoint) Path() string {
	return "/metrics/system"
}

func (m *SystemMetricsEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method: "GET",
			Func:   m.Get,
		},
	}
}

func (m *SystemMetricsEndpoint) Get(w http.ResponseWriter, r *http.Request) {
	systemJson, err := json.Marshal(metrics.Global.System)
	if err != nil {
		slog.Error("Error marshaling system metrics", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(systemJson)
}

package metrics

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reservoir/metrics"
	"reservoir/webserver/api/apitypes"
)

type TimingMetricsEndpoint struct{}

func (m *TimingMetricsEndpoint) Path() string {
	return "/metrics/timing"
}

func (m *TimingMetricsEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method: "GET",
			Func:   m.Get,
		},
	}
}

func (m *TimingMetricsEndpoint) Get(w http.ResponseWriter, r *http.Request) {
	timingJson, err := json.Marshal(metrics.Global.Timing)
	if err != nil {
		slog.Error("Error marshaling timing metrics", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(timingJson)
}

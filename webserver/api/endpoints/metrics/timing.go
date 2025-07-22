package metrics

import (
	"apt_cacher_go/metrics"
	"apt_cacher_go/webserver/api/api_types"
	"encoding/json"
	"log"
	"net/http"
)

type TimingMetricsEndpoint struct{}

func (m *TimingMetricsEndpoint) Path() string {
	return "/metrics/timing"
}

func (m *TimingMetricsEndpoint) EndpointMethods() []api_types.EndpointMethod {
	return []api_types.EndpointMethod{
		{
			Method: "GET",
			Func:   m.Get,
		},
	}
}

func (m *TimingMetricsEndpoint) Get(w http.ResponseWriter, r *http.Request) {
	timingJson, err := json.Marshal(metrics.Global.Timing)
	if err != nil {
		log.Printf("Error marshaling timing metrics: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(timingJson)
}

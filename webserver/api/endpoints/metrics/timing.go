package metrics

import (
	"apt_cacher_go/metrics"
	"encoding/json"
	"log"
	"net/http"
)

type GetTimingMetricsEndpoint struct{}

func (m *GetTimingMetricsEndpoint) Path() string {
	return "/metrics/timing"
}

func (m *GetTimingMetricsEndpoint) Method() string {
	return "GET"
}

func (m *GetTimingMetricsEndpoint) Endpoint(w http.ResponseWriter, r *http.Request) {
	timingJson, err := json.Marshal(metrics.Global.Timing)
	if err != nil {
		log.Printf("Error marshaling timing metrics: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(timingJson)
}

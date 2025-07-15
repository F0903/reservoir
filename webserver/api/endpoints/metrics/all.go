package metrics

import (
	"apt_cacher_go/metrics"
	"encoding/json"
	"log"
	"net/http"
)

type GetAllMetricsEndpoint struct{}

func (m *GetAllMetricsEndpoint) Path() string {
	return "/metrics"
}

func (m *GetAllMetricsEndpoint) Method() string {
	return "GET"
}

func (m *GetAllMetricsEndpoint) Endpoint(w http.ResponseWriter, r *http.Request) {
	metricsJson, err := json.Marshal(metrics.Global)
	if err != nil {
		log.Printf("Error marshaling all metrics: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(metricsJson)
}

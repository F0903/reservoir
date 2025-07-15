package metrics

import (
	"apt_cacher_go/metrics"
	"encoding/json"
	"log"
	"net/http"
)

type GetRequestsMetricsEndpoint struct{}

func (m *GetRequestsMetricsEndpoint) Path() string {
	return "/metrics/requests"
}

func (m *GetRequestsMetricsEndpoint) Method() string {
	return "GET"
}

func (m *GetRequestsMetricsEndpoint) Endpoint(w http.ResponseWriter, r *http.Request) {
	requestsJson, err := json.Marshal(metrics.Global.Requests)
	if err != nil {
		log.Printf("Error marshaling requests metrics: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(requestsJson)
}

package metrics

import (
	"apt_cacher_go/metrics"
	"apt_cacher_go/webserver/api/api_types"
	"encoding/json"
	"log"
	"net/http"
)

type AllMetricsEndpoint struct{}

func (m *AllMetricsEndpoint) Path() string {
	return "/metrics"
}

func (m *AllMetricsEndpoint) EndpointMethods() []api_types.EndpointMethod {
	return []api_types.EndpointMethod{
		{
			Method: "GET",
			Func:   m.Get,
		},
	}
}

func (m *AllMetricsEndpoint) Get(w http.ResponseWriter, r *http.Request) {
	metricsJson, err := json.Marshal(metrics.Global)
	if err != nil {
		log.Printf("Error marshaling all metrics: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(metricsJson)
}

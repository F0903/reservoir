package metrics

import (
	"apt_cacher_go/metrics"
	"apt_cacher_go/webserver/api/api_types"
	"encoding/json"
	"log"
	"net/http"
)

type RequestsMetricsEndpoint struct{}

func (m *RequestsMetricsEndpoint) Path() string {
	return "/metrics/requests"
}

func (m *RequestsMetricsEndpoint) EndpointMethods() []api_types.EndpointMethod {
	return []api_types.EndpointMethod{
		{
			Method: "GET",
			Func:   m.Get,
		},
	}
}

func (m *RequestsMetricsEndpoint) Get(w http.ResponseWriter, r *http.Request) {
	requestsJson, err := json.Marshal(metrics.Global.Requests)
	if err != nil {
		log.Printf("Error marshaling requests metrics: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(requestsJson)
}

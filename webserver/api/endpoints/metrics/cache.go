package metrics

import (
	"apt_cacher_go/metrics"
	"apt_cacher_go/webserver/api/api_types"
	"encoding/json"
	"log"
	"net/http"
)

type CacheMetricsEndpoint struct{}

func (m *CacheMetricsEndpoint) Path() string {
	return "/metrics/cache"
}

func (m *CacheMetricsEndpoint) EndpointMethods() []api_types.EndpointMethod {
	return []api_types.EndpointMethod{
		{
			Method: "GET",
			Func:   m.Get,
		},
	}
}

func (m *CacheMetricsEndpoint) Get(w http.ResponseWriter, r *http.Request) {
	cacheJson, err := json.Marshal(metrics.Global.Cache)
	if err != nil {
		log.Printf("Error marshaling cache metrics: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(cacheJson)
}

package metrics

import (
	"apt_cacher_go/metrics"
	"encoding/json"
	"log"
	"net/http"
)

type GetCacheMetricsEndpoint struct{}

func (m *GetCacheMetricsEndpoint) Path() string {
	return "/metrics/cache"
}

func (m *GetCacheMetricsEndpoint) Method() string {
	return "GET"
}

func (m *GetCacheMetricsEndpoint) Endpoint(w http.ResponseWriter, r *http.Request) {
	cacheJson, err := json.Marshal(metrics.Global.Cache)
	if err != nil {
		log.Printf("Error marshaling cache metrics: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(cacheJson)
}

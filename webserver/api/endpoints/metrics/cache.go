package metrics

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reservoir/metrics"
	"reservoir/webserver/api/apitypes"
)

type CacheMetricsEndpoint struct{}

func (m *CacheMetricsEndpoint) Path() string {
	return "/metrics/cache"
}

func (m *CacheMetricsEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method: "GET",
			Func:   m.Get,
		},
	}
}

func (m *CacheMetricsEndpoint) Get(w http.ResponseWriter, r *http.Request) {
	cacheJson, err := json.Marshal(metrics.Global.Cache)
	if err != nil {
		slog.Error("Error marshaling cache metrics", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(cacheJson)
}

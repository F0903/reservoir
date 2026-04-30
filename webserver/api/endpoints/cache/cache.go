package cache

import (
	"log/slog"
	"net/http"
	"reservoir/config"
	"reservoir/webserver/api/apihttp"
	"reservoir/webserver/api/apitypes"
)

type StatusEndpoint struct{}

func (e *StatusEndpoint) Path() string {
	return "/cache/status"
}

func (e *StatusEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:       http.MethodGet,
			Func:         e.Get,
			RequiresAuth: true,
		},
	}
}

type statusResponse struct {
	Type           config.CacheType `json:"type"`
	Entries        int              `json:"entries"`
	Bytes          int64            `json:"bytes"`
	MaxBytes       int64            `json:"max_bytes"`
	MemoryCapBytes *int64           `json:"memory_cap_bytes,omitempty"`
}

func (e *StatusEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	if !requireCacheController(w, ctx) {
		return
	}

	stats := ctx.Cache.CacheStats()
	cacheType := ctx.Config.Cache.Type.Read()

	resp := statusResponse{
		Type:     cacheType,
		Entries:  stats.Entries,
		Bytes:    stats.Bytes,
		MaxBytes: stats.MaxBytes,
	}
	if cacheType == config.CacheTypeMemory {
		resp.MemoryCapBytes = &stats.MemoryCapBytes
	}

	apihttp.WriteJSON(w, http.StatusOK, resp)
}

type ClearEndpoint struct{}

func (e *ClearEndpoint) Path() string {
	return "/cache/clear"
}

func (e *ClearEndpoint) EndpointMethods() []apitypes.EndpointMethod {
	return []apitypes.EndpointMethod{
		{
			Method:       http.MethodPost,
			Func:         e.Post,
			RequiresAuth: true,
		},
	}
}

func (e *ClearEndpoint) Post(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	if !requireCacheController(w, ctx) {
		return
	}

	if err := ctx.Cache.ClearCache(); err != nil {
		slog.Error("Failed to clear cache", "error", err)
		apihttp.InternalServerError(w)
		return
	}

	apihttp.NoContent(w)
}

func requireCacheController(w http.ResponseWriter, ctx apitypes.Context) bool {
	if ctx.Cache != nil {
		return true
	}

	slog.Error("Cache endpoint requested without a cache controller")
	apihttp.Error(w, "Cache unavailable", http.StatusServiceUnavailable)
	return false
}

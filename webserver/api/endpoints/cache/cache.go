package cache

import (
	"log/slog"
	"net/http"
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
	Type           string `json:"type"`
	Entries        int    `json:"entries"`
	Bytes          int64  `json:"bytes"`
	MaxBytes       int64  `json:"max_bytes"`
	MemoryCapBytes *int64 `json:"memory_cap_bytes,omitempty"`
}

func (e *StatusEndpoint) Get(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	if !requireCacheController(w, ctx) {
		return
	}

	status, err := ctx.Services.Cache.Status()
	if err != nil {
		slog.Error("Failed to get cache status", "error", err)
		apihttp.InternalServerError(w)
		return
	}

	resp := statusResponse{
		Type:           string(status.Type),
		Entries:        status.Entries,
		Bytes:          status.Bytes,
		MaxBytes:       status.MaxBytes,
		MemoryCapBytes: status.MemoryCapBytes,
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
			Method:        http.MethodPost,
			Func:          e.Post,
			RequiresAuth:  true,
			RequiresAdmin: true,
		},
	}
}

func (e *ClearEndpoint) Post(w http.ResponseWriter, r *http.Request, ctx apitypes.Context) {
	if !requireCacheController(w, ctx) {
		return
	}

	if err := ctx.Services.Cache.Clear(); err != nil {
		slog.Error("Failed to clear cache", "error", err)
		apihttp.InternalServerError(w)
		return
	}

	apihttp.NoContent(w)
}

func requireCacheController(w http.ResponseWriter, ctx apitypes.Context) bool {
	if ctx.Services != nil && ctx.Services.Cache != nil && ctx.Services.Cache.Available() {
		return true
	}

	slog.Error("Cache endpoint requested without a cache controller")
	apihttp.Error(w, "Cache unavailable", http.StatusServiceUnavailable)
	return false
}

package metrics

import (
	runtimeMetrics "reservoir/metrics"
	"reservoir/webserver/api/apitypes"
)

func metricsFromContext(ctx apitypes.Context) (*runtimeMetrics.Metrics, bool) {
	if ctx.Services == nil || ctx.Services.Metrics == nil {
		return nil, false
	}
	return ctx.Services.Metrics, true
}

func collectCacheStorage(ctx apitypes.Context, m *runtimeMetrics.Metrics) {
	if m == nil {
		return
	}

	if ctx.Services == nil || ctx.Services.Cache == nil {
		m.Cache.SetStorage(runtimeMetrics.CacheStorageMetrics{})
		return
	}

	storage, ok := ctx.Services.Cache.StorageMetrics()
	if !ok {
		m.Cache.SetStorage(runtimeMetrics.CacheStorageMetrics{})
		return
	}

	m.Cache.SetStorage(storage)
}

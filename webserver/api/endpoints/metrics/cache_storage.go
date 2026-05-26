package metrics

import (
	runtimeMetrics "reservoir/metrics"
	"reservoir/webserver/api/apitypes"
)

func collectCacheStorage(ctx apitypes.Context) {
	if ctx.Services == nil || ctx.Services.Cache == nil {
		runtimeMetrics.Global.Cache.SetStorage(runtimeMetrics.CacheStorageMetrics{})
		return
	}

	storage, ok := ctx.Services.Cache.StorageMetrics()
	if !ok {
		runtimeMetrics.Global.Cache.SetStorage(runtimeMetrics.CacheStorageMetrics{})
		return
	}

	runtimeMetrics.Global.Cache.SetStorage(storage)
}

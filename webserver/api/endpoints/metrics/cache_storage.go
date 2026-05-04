package metrics

import (
	"reservoir/config"
	runtimeMetrics "reservoir/metrics"
	"reservoir/webserver/api/apitypes"
)

func collectCacheStorage(ctx apitypes.Context) {
	if ctx.Cache == nil || ctx.Config == nil {
		runtimeMetrics.Global.Cache.SetStorage(runtimeMetrics.CacheStorageMetrics{})
		return
	}

	stats := ctx.Cache.CacheStats()
	cacheType := ctx.Config.Cache.Type.Read()
	storage := runtimeMetrics.CacheStorageMetrics{
		Type:     string(cacheType),
		Entries:  stats.Entries,
		Bytes:    stats.Bytes,
		MaxBytes: stats.MaxBytes,
	}

	if cacheType == config.CacheTypeMemory || cacheType == config.CacheTypeHybrid {
		memoryCapBytes := stats.MemoryCapBytes
		storage.MemoryCapBytes = &memoryCapBytes
	}

	runtimeMetrics.Global.Cache.SetStorage(storage)
}

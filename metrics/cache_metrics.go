package metrics

import "reservoir/utils/atomics"

type cacheMetrics struct {
	CacheHits      atomics.Int64 `json:"cache_hits"`
	CacheMisses    atomics.Int64 `json:"cache_misses"`
	CacheErrors    atomics.Int64 `json:"cache_errors"`
	CacheEntries   atomics.Int64 `json:"cache_entries"`
	BytesCached    atomics.Int64 `json:"bytes_cached"`
	CleanupRuns    atomics.Int64 `json:"cleanup_runs"`
	BytesCleaned   atomics.Int64 `json:"bytes_cleaned"`
	CacheEvictions atomics.Int64 `json:"cache_evictions"`
}

func NewCacheMetrics() cacheMetrics {
	return cacheMetrics{
		CacheHits:      atomics.NewInt64(0),
		CacheMisses:    atomics.NewInt64(0),
		CacheErrors:    atomics.NewInt64(0),
		CacheEntries:   atomics.NewInt64(0),
		BytesCached:    atomics.NewInt64(0),
		CleanupRuns:    atomics.NewInt64(0),
		BytesCleaned:   atomics.NewInt64(0),
		CacheEvictions: atomics.NewInt64(0),
	}
}

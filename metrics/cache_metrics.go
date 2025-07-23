package metrics

import "apt_cacher_go/utils/atomics"

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
	// Since Go always zero-initializes structs, we can just return a new "empty" instance.
	return cacheMetrics{}
}

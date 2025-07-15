package metrics

type cacheMetrics struct {
	CacheHits    AtomicInt64 `json:"cache_hits"`
	CacheMisses  AtomicInt64 `json:"cache_misses"`
	CacheErrors  AtomicInt64 `json:"cache_errors"`
	CacheEntries AtomicInt64 `json:"cache_entries"`
	BytesCached  AtomicInt64 `json:"bytes_cached"`
}

func NewCacheMetrics() cacheMetrics {
	// Since Go always zero-initializes structs, we can just return a new "empty" instance.
	return cacheMetrics{}
}

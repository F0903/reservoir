package metrics

type cacheMetrics struct {
	CacheHits    AtomicInt64
	CacheMisses  AtomicInt64
	CacheErrors  AtomicInt64
	CacheEntries AtomicInt64
	BytesCached  AtomicInt64
}

func NewCacheMetrics() cacheMetrics {
	// Since Go always zero-initializes structs, we can just return a new "empty" instance.
	return cacheMetrics{}
}

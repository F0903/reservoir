package cache

import (
	"reservoir/metrics"
	"reservoir/utils"
	"reservoir/utils/atomics"
	"sync"
)

// Gets or creates a lock for the given key.
func GetLock(locks []sync.RWMutex, key CacheKey) *sync.RWMutex {
	val := utils.Hex8ToIndex(key.Hex)
	return &locks[val%uint32(len(locks))]
}

func AddCacheSize(m *metrics.Metrics, byteCounter *atomics.Int64, delta int64) {
	byteCounter.Add(delta)
	metrics.Ensure(m).Cache.BytesCached.Add(delta)
}

func DecrementCacheSize(m *metrics.Metrics, byteCounter *atomics.Int64, delta int64) {
	byteCounter.Sub(delta)
	metrics.Ensure(m).Cache.BytesCached.Sub(delta)
}

func IncrementCacheEntries(m *metrics.Metrics) {
	metrics.Ensure(m).Cache.CacheEntries.Add(1)
}

func DecrementCacheEntries(m *metrics.Metrics) {
	metrics.Ensure(m).Cache.CacheEntries.Sub(1)
}

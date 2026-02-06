package cache

import (
	"reservoir/metrics"
	"reservoir/utils"
	"reservoir/utils/atomics"
	"sync"
)

// Gets or creates a lock for the given key.
func getLock(locks *[CACHE_LOCK_SHARDS]sync.RWMutex, key CacheKey) *sync.RWMutex {
	val := utils.Hex8ToIndex(key.Hex)
	return &locks[val%CACHE_LOCK_SHARDS]
}

func addCacheSize(byteCounter *atomics.Int64, delta int64) {
	byteCounter.Add(delta)
	metrics.Global.Cache.BytesCached.Add(delta)
}

func decrementCacheSize(byteCounter *atomics.Int64, delta int64) {
	byteCounter.Sub(delta)
	metrics.Global.Cache.BytesCached.Sub(delta)
}

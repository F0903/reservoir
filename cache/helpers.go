package cache

import (
	"reservoir/metrics"
	"reservoir/utils"
	"reservoir/utils/atomics"
	"sync"
)

// Gets or creates a lock for the given key.
func getLock(locks []sync.RWMutex, key CacheKey) *sync.RWMutex {
	val := utils.Hex8ToIndex(key.Hex)
	return &locks[val%uint32(len(locks))]
}

func addCacheSize(byteCounter *atomics.Int64, delta int64) {
	byteCounter.Add(delta)
	metrics.Global.Cache.BytesCached.Add(delta)
}

func decrementCacheSize(byteCounter *atomics.Int64, delta int64) {
	byteCounter.Sub(delta)
	metrics.Global.Cache.BytesCached.Sub(delta)
}

func incrementCacheEntries() {
	metrics.Global.Cache.CacheEntries.Add(1)
}

func decrementCacheEntries() {
	metrics.Global.Cache.CacheEntries.Sub(1)
}

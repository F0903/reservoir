package cache

import (
	"cmp"
	"iter"
	"log/slog"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/utils/bytesize"
	"slices"
	"sync"
	"time"
)

type cacheJanitor struct {
	stopChan chan struct{}
	interval time.Duration
	running  bool

	cacheIterator func() iter.Seq2[CacheKey, *EntryMetadata[any]]
	removeEntry   func(key CacheKey) error
	getCacheSize  func() int64
	getCacheLen   func() int
	getLock       func(key CacheKey) *sync.RWMutex
}

func newCacheJanitor(stopChan chan struct{}, interval time.Duration) *cacheJanitor {
	return &cacheJanitor{
		stopChan: stopChan,
		interval: interval,
	}
}

func (j *cacheJanitor) start() {
	if j.running {
		return
	}
	j.running = true

	go func() {
		ticker := time.NewTicker(j.interval)
		defer ticker.Stop()

		slog.Info("Cache cleanup task started")
		for {
			select {
			case <-ticker.C:
				slog.Info("Running cache cleanup cycle...")
				j.cleanExpiredEntries()
				j.ensureCacheSize()
				metrics.Global.Cache.CleanupRuns.Increment()
				slog.Info("Cache cleanup cycle complete")
			case <-j.stopChan:
				slog.Info("Cache cleanup task stopped")
				return
			}
		}
	}()
}

func (j *cacheJanitor) stop() {
	if !j.running {
		return
	}
	j.running = false

	close(j.stopChan)
}

func (j *cacheJanitor) cleanExpiredEntries() {
	slog.Info("Cleaning up expired cache entries")

	startCacheSize := j.getCacheSize()

	keysToRemove := make([]CacheKey, 0)

	for key, meta := range j.cacheIterator() {
		expired := meta.Expires.Before(time.Now())

		if !expired {
			continue
		}

		slog.Info("Found expired cache entry for key", "key", key.Hex)
		keysToRemove = append(keysToRemove, key)
	}

	for _, key := range keysToRemove {
		slog.Info("Removing expired cache entry for key", "key", key.Hex)

		lock := j.getLock(key)
		locked := lock.TryLock()
		if !locked {
			slog.Info("Failed to acquire lock for key", "key", key.Hex)
			continue
		}

		if err := j.removeEntry(key); err != nil {
			lock.Unlock()
			slog.Info("Failed to remove expired cache entry for key", "key", key.Hex, "error", err)
			continue
		}
		lock.Unlock()

		slog.Info("Removed expired cache entry for key", "key", key.Hex)
	}

	endCacheSize := j.getCacheSize()
	metrics.Global.Cache.BytesCached.Set(endCacheSize)
	metrics.Global.Cache.BytesCleaned.Add(startCacheSize - endCacheSize)

	slog.Info("Cache cleanup complete", "new_size", endCacheSize)
}

func (j *cacheJanitor) ensureCacheSize() {
	maxCacheSize := config.Global.MaxCacheSize.Read().Bytes()
	startCacheSize := j.getCacheSize()
	if startCacheSize < maxCacheSize {
		return
	}

	slog.Info("Cache size exceeds limit, starting eviction", "byte_size", startCacheSize, "max_cache_size", maxCacheSize)

	type entryForEviction struct {
		key      CacheKey
		meta     *EntryMetadata[any]
		priority int64 // Lower = evict first
	}

	candidates := make([]entryForEviction, 0, j.getCacheLen())
	now := time.Now()

	for key, meta := range j.cacheIterator() {
		timeSinceAccess := now.Sub(meta.LastAccess).Milliseconds()
		sizeWeight := meta.Size / bytesize.UnitM

		// Calculate eviction priority (highest = evict first)
		// Factors: age since last access + file size weight
		priority := timeSinceAccess + (sizeWeight * 100) // Give size significant weight

		candidates = append(candidates, entryForEviction{
			key:      key,
			meta:     meta,
			priority: priority,
		})
	}

	// Sort by priority (highest = evict first)
	slices.SortFunc(candidates, func(x, y entryForEviction) int {
		return cmp.Compare(y.priority, x.priority) // Swapped x and y for descending order
	})

	// Evict entries until we're under the limit
	targetSize := int64(float64(maxCacheSize) * 0.8) // Evict to 80% to avoid thrashing

	slog.Info("Target size for eviction", "target_size", bytesize.ByteSize(targetSize))
	evictions := 0
	for _, candidate := range candidates {
		if j.getCacheSize() <= targetSize {
			break
		}

		lock := j.getLock(candidate.key)
		if lock.TryLock() {
			slog.Info("Evicting cache entry", "key", candidate.key.Hex, "size", candidate.meta.Size, "last_access", candidate.meta.LastAccess)

			if err := j.removeEntry(candidate.key); err != nil {
				slog.Info("Failed to evict cache entry", "key", candidate.key.Hex, "error", err)
			}
			evictions++
			lock.Unlock()
		} else {
			slog.Info("Failed to acquire lock for cache entry", "key", candidate.key.Hex)
			continue
		}
	}

	endCacheSize := j.getCacheSize()
	metrics.Global.Cache.BytesCached.Set(endCacheSize)
	metrics.Global.Cache.BytesCleaned.Add(startCacheSize - endCacheSize)

	slog.Info("Cache eviction complete", "evicted_entries", evictions, "new_size", endCacheSize)
}

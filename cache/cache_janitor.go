package cache

import (
	"cmp"
	"context"
	"iter"
	"log/slog"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/utils/bytesize"
	"reservoir/utils/duration"
	"slices"
	"sync"
	"time"
)

type cacheFunctions[MetadataT any] struct {
	cacheIterator iter.Seq2[CacheKey, *EntryMetadata[MetadataT]]
	removeEntry   func(key CacheKey) error
	getCacheSize  func() int64
	getCacheLen   func() int
	getLock       func(key CacheKey) *sync.RWMutex
}

type cacheJanitor[MetadataT any] struct {
	stopChan        chan struct{}
	intervalChanged chan time.Duration
	interval        time.Duration
	running         bool

	cacheFns cacheFunctions[MetadataT]
	subs     config.ConfigSubscriber
}

func newCacheJanitor[MetadataT any](interval time.Duration, cacheFns cacheFunctions[MetadataT]) *cacheJanitor[MetadataT] {
	j := &cacheJanitor[MetadataT]{
		stopChan:        make(chan struct{}),
		intervalChanged: make(chan time.Duration, 1),
		interval:        interval,
		cacheFns:        cacheFns,
	}

	j.subs.Add(config.Global.Cache.CleanupInterval.OnChange(func(newInterval duration.Duration) {
		slog.Info("Cache cleanup interval changed", "new_interval", newInterval)
		j.intervalChanged <- newInterval.Cast()
	}))

	return j
}

func (j *cacheJanitor[MetadataT]) start(ctx context.Context) {
	if j.running {
		return
	}
	j.running = true

	go func() {
		ticker := time.NewTicker(j.interval)
		defer ticker.Stop()

		slog.Info("Cache cleanup task started", "interval", j.interval)
		for {
			select {
			case <-ticker.C:
				slog.Info("Running cache cleanup cycle...")
				j.cleanExpiredEntries()
				j.ensureCacheSize()
				metrics.Global.Cache.CleanupRuns.Increment()
				slog.Info("Cache cleanup cycle complete")
			case newInterval := <-j.intervalChanged:
				j.interval = newInterval
				ticker.Reset(j.interval)
				slog.Info("Cache cleanup ticker reset", "new_interval", j.interval)
			case <-j.stopChan:
				slog.Info("Cache cleanup task stopped")
				return
			case <-ctx.Done():
				slog.Info("Cache cleanup task stopped")
				return
			}
		}
	}()
}

func (j *cacheJanitor[MetadataT]) stop() {
	if !j.running {
		return
	}
	j.running = false

	j.subs.UnsubscribeAll()

	close(j.stopChan)
}

func (j *cacheJanitor[MetadataT]) cleanExpiredEntries() {
	slog.Info("Cleaning up expired cache entries")

	startCacheSize := j.cacheFns.getCacheSize()

	keysToRemove := make([]CacheKey, 0)

	for key, meta := range j.cacheFns.cacheIterator {
		expired := meta.Expires.Before(time.Now())

		if !expired {
			continue
		}

		slog.Info("Found expired cache entry for key", "key", key.Hex)
		keysToRemove = append(keysToRemove, key)
	}

	for _, key := range keysToRemove {
		slog.Info("Removing expired cache entry for key", "key", key.Hex)

		lock := j.cacheFns.getLock(key)
		locked := lock.TryLock()
		if !locked {
			slog.Info("Failed to acquire lock for key", "key", key.Hex)
			continue
		}

		if err := j.cacheFns.removeEntry(key); err != nil {
			lock.Unlock()
			slog.Info("Failed to remove expired cache entry for key", "key", key.Hex, "error", err)
			continue
		}
		lock.Unlock()

		slog.Info("Removed expired cache entry for key", "key", key.Hex)
	}

	endCacheSize := j.cacheFns.getCacheSize()
	metrics.Global.Cache.BytesCached.Set(endCacheSize)
	metrics.Global.Cache.BytesCleaned.Add(startCacheSize - endCacheSize)

	slog.Info("Cache cleanup complete", "new_size", endCacheSize)
}

// Evict entries until 80% of maxCacheBytes is reached
func (j *cacheJanitor[MetadataT]) evict(maxCacheBytes int64) {
	type entryForEviction struct {
		key      CacheKey
		meta     *EntryMetadata[MetadataT]
		priority int64 // Lower = evict first
	}

	candidates := make([]entryForEviction, 0, j.cacheFns.getCacheLen())
	now := time.Now()

	for key, meta := range j.cacheFns.cacheIterator {
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
	targetSize := int64(float64(maxCacheBytes) * 0.8) // Evict to 80% to avoid thrashing

	startCacheSize := j.cacheFns.getCacheSize()

	slog.Info("Target size for eviction", "target_size", bytesize.ByteSize(targetSize))
	evictions := 0
	for _, candidate := range candidates {
		if j.cacheFns.getCacheSize() <= targetSize {
			break
		}

		lock := j.cacheFns.getLock(candidate.key)
		if lock.TryLock() {
			slog.Info("Evicting cache entry", "key", candidate.key.Hex, "size", candidate.meta.Size, "last_access", candidate.meta.LastAccess)

			if err := j.cacheFns.removeEntry(candidate.key); err != nil {
				slog.Info("Failed to evict cache entry", "key", candidate.key.Hex, "error", err)
			} else {
				metrics.Global.Cache.CacheEvictions.Increment()
			}
			evictions++
			lock.Unlock()
		} else {
			slog.Info("Failed to acquire lock for cache entry", "key", candidate.key.Hex)
			continue
		}
	}

	endCacheSize := j.cacheFns.getCacheSize()
	metrics.Global.Cache.BytesCached.Set(endCacheSize)
	metrics.Global.Cache.BytesCleaned.Add(startCacheSize - endCacheSize)

	slog.Info("Cache eviction complete", "evicted_entries", evictions, "new_size", endCacheSize)
}

func (j *cacheJanitor[MetadataT]) ensureCacheSize() {
	maxCacheSize := config.Global.Cache.MaxCacheSize.Read().Bytes()
	startCacheSize := j.cacheFns.getCacheSize()
	if startCacheSize < maxCacheSize {
		return
	}

	slog.Info("Cache size exceeds limit, starting eviction", "byte_size", startCacheSize, "max_cache_size", maxCacheSize)

	j.evict(maxCacheSize)
}

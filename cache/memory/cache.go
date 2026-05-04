package memory

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"reservoir/cache"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/utils/atomics"
	"reservoir/utils/bytesize"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/mem"
)

type Cache[MetadataT any] struct {
	entries      map[cache.CacheKey]*memoryInternalEntry[MetadataT]
	mu           sync.RWMutex
	locks        []sync.RWMutex
	memoryCap    atomics.Int64
	maxCacheSize atomics.Int64
	byteSize     atomics.Int64

	janitor *cache.Janitor[MetadataT]
	subs    config.ConfigSubscriber
}

func New[MetadataT any](cfg *config.Config, memoryBudgetPercent int, maxCacheSize int64, cleanupInterval time.Duration, shardCount int, ctx context.Context) *Cache[MetadataT] {
	return newWithAggregateMetrics[MetadataT](cfg, memoryBudgetPercent, maxCacheSize, cleanupInterval, shardCount, ctx, true)
}

func NewTier[MetadataT any](cfg *config.Config, memoryBudgetPercent int, maxCacheSize int64, cleanupInterval time.Duration, shardCount int, ctx context.Context) *Cache[MetadataT] {
	return newWithAggregateMetrics[MetadataT](cfg, memoryBudgetPercent, maxCacheSize, cleanupInterval, shardCount, ctx, false)
}

func newWithAggregateMetrics[MetadataT any](cfg *config.Config, memoryBudgetPercent int, maxCacheSize int64, cleanupInterval time.Duration, shardCount int, ctx context.Context, trackAggregateMetrics bool) *Cache[MetadataT] {
	sysMem, err := mem.VirtualMemory()
	if err != nil {
		panic(fmt.Sprintf("failed to get system memory info: %v", err))
	}

	c := &Cache[MetadataT]{
		entries:      make(map[cache.CacheKey]*memoryInternalEntry[MetadataT]),
		locks:        make([]sync.RWMutex, shardCount),
		memoryCap:    atomics.NewInt64(int64(sysMem.Total) * int64(memoryBudgetPercent) / 100),
		maxCacheSize: atomics.NewInt64(maxCacheSize),
		byteSize:     atomics.NewInt64(0),
	}

	c.subs.Add(cfg.Cache.MaxCacheSize.OnChange(func(newSize bytesize.ByteSize) {
		c.maxCacheSize.Set(newSize.Bytes())
	}))

	c.subs.Add(cfg.Cache.Memory.MemoryBudgetPercent.OnChange(func(newPercent int) {
		newCap := int64(sysMem.Total) * int64(newPercent) / 100
		c.memoryCap.Set(newCap)
		slog.Info("Memory budget changed", "new_percent", newPercent, "new_cap", bytesize.ByteSize(newCap))
	}))

	c.janitor = cache.NewJanitor(cfg, cleanupInterval, cache.JanitorFunctions[MetadataT]{
		Iterate: func(yield func(key cache.CacheKey, metadata *cache.EntryMetadata[MetadataT]) bool) {
			c.mu.RLock()
			snapshot := maps.Clone(c.entries)
			c.mu.RUnlock()

			for key := range snapshot {
				lock := cache.GetLock(c.locks, key)
				lock.RLock()
				c.mu.RLock()
				entry, ok := c.entries[key]
				c.mu.RUnlock()

				var meta *cache.EntryMetadata[MetadataT]
				if ok {
					meta = entry.metadataSnapshot()
				}
				lock.RUnlock()

				if !ok {
					continue
				}
				if !yield(key, meta) {
					break
				}
			}
		},
		Remove: func(key cache.CacheKey) error {
			return c.deleteInternal(key)
		},
		Size: func() int64 {
			return c.byteSize.Get()
		},
		Len: func() int {
			c.mu.RLock()
			defer c.mu.RUnlock()
			return len(c.entries)
		},
		Lock: func(key cache.CacheKey) *sync.RWMutex {
			return cache.GetLock(c.locks, key)
		},
	}, trackAggregateMetrics)
	c.janitor.Start(ctx)

	return c
}

func (c *Cache[MetadataT]) recordCacheHit() {
	metrics.Global.Cache.CacheHits.Increment()
}

func (c *Cache[MetadataT]) recordCacheMiss() {
	metrics.Global.Cache.CacheMisses.Increment()
}

func (c *Cache[MetadataT]) recordCacheError() {
	metrics.Global.Cache.CacheErrors.Increment()
}

func (c *Cache[MetadataT]) Destroy() {
	c.janitor.Stop()
	c.subs.UnsubscribeAll()
}

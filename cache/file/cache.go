package file

import (
	"context"
	"maps"
	"path/filepath"
	"reservoir/cache"
	"reservoir/cache/internal/tier"
	"reservoir/config"
	"reservoir/utils/assertedpath"
	"reservoir/utils/atomics"
	"reservoir/utils/bytesize"
	"sync"
	"time"
)

type Cache[MetadataT any] struct {
	rootDir         assertedpath.AssertedPath
	entriesMetadata map[cache.CacheKey]*cache.EntryMetadata[MetadataT]
	mu              sync.RWMutex
	locks           []sync.RWMutex
	byteSize        atomics.Int64
	maxCacheSize    atomics.Int64
	janitor         *tier.Janitor[MetadataT]
	subs            config.ConfigSubscriber
}

func New[MetadataT any](cfg *config.Config, rootDir string, maxCacheSize int64, cleanupInterval time.Duration, shardCount int, ctx context.Context) *Cache[MetadataT] {
	return newWithAggregateMetrics[MetadataT](cfg, rootDir, maxCacheSize, cleanupInterval, shardCount, ctx, true)
}

func NewTier[MetadataT any](cfg *config.Config, rootDir string, maxCacheSize int64, cleanupInterval time.Duration, shardCount int, ctx context.Context) *Cache[MetadataT] {
	return newWithAggregateMetrics[MetadataT](cfg, rootDir, maxCacheSize, cleanupInterval, shardCount, ctx, false)
}

func newWithAggregateMetrics[MetadataT any](cfg *config.Config, rootDir string, maxCacheSize int64, cleanupInterval time.Duration, shardCount int, ctx context.Context, trackAggregateMetrics bool) *Cache[MetadataT] {
	c := &Cache[MetadataT]{
		rootDir:         assertedpath.AssertDirectory(rootDir),
		entriesMetadata: make(map[cache.CacheKey]*cache.EntryMetadata[MetadataT]),
		locks:           make([]sync.RWMutex, shardCount),
		byteSize:        atomics.NewInt64(0),
		maxCacheSize:    atomics.NewInt64(maxCacheSize),
	}

	c.subs.Add(cfg.Cache.MaxCacheSize.OnChange(func(newSize bytesize.ByteSize) {
		c.maxCacheSize.Set(newSize.Bytes())
	}))

	c.loadMetadataSidecars()

	c.janitor = tier.NewJanitor(cfg, cleanupInterval, tier.Functions[MetadataT]{
		Iterate: func(yield func(key cache.CacheKey, metadata *cache.EntryMetadata[MetadataT]) bool) {
			c.mu.RLock()
			snapshot := maps.Clone(c.entriesMetadata)
			c.mu.RUnlock()

			for key, metadata := range snapshot {
				if !yield(key, metadata) {
					break
				}
			}
		},
		Size: func() int64 {
			return c.byteSize.Get()
		},
		Len: func() int {
			c.mu.RLock()
			defer c.mu.RUnlock()
			return len(c.entriesMetadata)
		},
		Remove: func(key cache.CacheKey) error {
			return c.ensureRemove(key)
		},
		Lock: func(key cache.CacheKey) *sync.RWMutex {
			return tier.GetLock(c.locks, key)
		},
	}, trackAggregateMetrics)
	if c.byteSize.Get() >= c.maxCacheSize.Get() {
		c.janitor.Evict(c.maxCacheSize.Get())
	}
	c.janitor.Start(ctx)
	return c
}

func (c *Cache[MetadataT]) Destroy() {
	c.janitor.Stop()
	c.subs.UnsubscribeAll()
}

func (c *Cache[MetadataT]) dataPath(key cache.CacheKey) string {
	return filepath.Join(c.rootDir.Path, key.Hex)
}

func (c *Cache[MetadataT]) metadataPath(key cache.CacheKey) string {
	return filepath.Join(c.rootDir.Path, key.Hex+".meta.json")
}

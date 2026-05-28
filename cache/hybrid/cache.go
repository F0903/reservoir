package hybrid

import (
	"context"
	"errors"
	"io"
	"reservoir/cache"
	filecache "reservoir/cache/file"
	memorycache "reservoir/cache/memory"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/utils/atomics"
	"reservoir/utils/bytesize"
	"reservoir/utils/duration"
	"sync"
	"time"
)

type Cache[MetadataT any] struct {
	file         *filecache.Cache[MetadataT]
	memory       *memorycache.Cache[MetadataT]
	maxCacheSize atomics.Int64
	demoteAfter  atomics.Int64
	placementMu  sync.Mutex

	stopDemoter chan struct{}
	demoterDone chan struct{}
	stopOnce    sync.Once
	subs        config.ConfigSubscriber
	metrics     *metrics.Metrics
}

func New[MetadataT any](cfg *config.Config, m *metrics.Metrics, rootDir string, memoryBudgetPercent int, maxCacheSize int64, cleanupInterval time.Duration, shardCount int, ctx context.Context) *Cache[MetadataT] {
	m = metrics.Ensure(m)
	c := &Cache[MetadataT]{
		file:         filecache.NewTier[MetadataT](cfg, m, rootDir, maxCacheSize, cleanupInterval, shardCount, ctx),
		memory:       memorycache.NewTier[MetadataT](cfg, m, memoryBudgetPercent, maxCacheSize, cleanupInterval, shardCount, ctx),
		maxCacheSize: atomics.NewInt64(maxCacheSize),
		demoteAfter:  atomics.NewInt64(int64(cfg.Cache.Hybrid.DemoteAfter.Read().Cast())),
		stopDemoter:  make(chan struct{}),
		demoterDone:  make(chan struct{}),
		metrics:      m,
	}

	c.subs.Add(cfg.Cache.MaxCacheSize.OnChange(func(newSize bytesize.ByteSize) {
		c.maxCacheSize.Set(newSize.Bytes())
		c.enforceMaxCacheSize()
	}))
	c.subs.Add(cfg.Cache.Hybrid.DemoteAfter.OnChange(func(newDuration duration.Duration) {
		c.demoteAfter.Set(int64(newDuration.Cast()))
	}))

	c.startDemoter(ctx)
	return c
}

func (c *Cache[MetadataT]) startDemoter(ctx context.Context) {
	go func() {
		defer close(c.demoterDone)

		for {
			demoteAfter := time.Duration(c.demoteAfter.Get())
			timer := time.NewTimer(demotionInterval(demoteAfter))

			select {
			case <-timer.C:
				c.demoteIdleEntries()
			case <-c.stopDemoter:
				timer.Stop()
				return
			case <-ctx.Done():
				timer.Stop()
				return
			}
		}
	}()
}

func (c *Cache[MetadataT]) Destroy() {
	c.stopOnce.Do(func() {
		close(c.stopDemoter)
		<-c.demoterDone
	})
	c.subs.UnsubscribeAll()
	c.memory.Destroy()
	c.file.Destroy()
}

func (c *Cache[MetadataT]) recordCacheHit() {
	c.metrics.Cache.CacheHits.Increment()
}

func (c *Cache[MetadataT]) recordCacheMiss() {
	c.metrics.Cache.CacheMisses.Increment()
}

func (c *Cache[MetadataT]) recordCacheError() {
	c.metrics.Cache.CacheErrors.Increment()
}

func (c *Cache[MetadataT]) Get(key cache.CacheKey) (*cache.Entry[MetadataT], error) {
	if entry, err := c.memory.GetQuiet(key); err == nil {
		c.recordCacheHit()
		return entry, nil
	} else if !errors.Is(err, cache.ErrCacheEntryNotFound) {
		c.recordCacheError()
		return nil, err
	}

	entry, err := c.file.GetQuiet(key)
	if err != nil {
		if errors.Is(err, cache.ErrCacheEntryNotFound) {
			c.recordCacheMiss()
		} else {
			c.recordCacheError()
		}
		return nil, err
	}

	c.recordCacheHit()
	if promoted, ok := c.promoteToMemory(key, entry); ok {
		return promoted, nil
	}
	return entry, nil
}

func (c *Cache[MetadataT]) Cache(key cache.CacheKey, data io.Reader, expires time.Time, metadata MetadataT) (*cache.Entry[MetadataT], error) {
	if sizeHint, ok := cache.ReaderSizeHint(data); ok {
		return c.cacheKnownSize(key, data, sizeHint, expires, metadata)
	}
	return c.cacheUnknownSize(key, data, expires, metadata)
}

func (c *Cache[MetadataT]) Delete(key cache.CacheKey) error {
	memoryErr := c.memory.Delete(key)
	fileErr := c.file.Delete(key)

	if errors.Is(memoryErr, cache.ErrCacheEntryNotFound) && errors.Is(fileErr, cache.ErrCacheEntryNotFound) {
		return cache.ErrCacheEntryNotFound
	}
	if errors.Is(memoryErr, cache.ErrCacheEntryNotFound) {
		memoryErr = nil
	}
	if errors.Is(fileErr, cache.ErrCacheEntryNotFound) {
		fileErr = nil
	}
	return errors.Join(memoryErr, fileErr)
}

func (c *Cache[MetadataT]) Stats() cache.Stats {
	memoryStats := c.memory.Stats()
	fileStats := c.file.Stats()

	return cache.Stats{
		Entries:        memoryStats.Entries + fileStats.Entries,
		Bytes:          memoryStats.Bytes + fileStats.Bytes,
		MaxBytes:       c.maxCacheSize.Get(),
		MemoryCapBytes: memoryStats.MemoryCapBytes,
	}
}

func (c *Cache[MetadataT]) Clear() error {
	memoryErr := c.memory.Clear()
	fileErr := c.file.Clear()

	if errors.Is(memoryErr, cache.ErrCacheEntryNotFound) {
		memoryErr = nil
	}
	return errors.Join(fileErr, memoryErr)
}

func (c *Cache[MetadataT]) getMetadata(key cache.CacheKey, recordMetrics bool) (meta *cache.EntryMetadata[MetadataT], stale bool, err error) {
	if meta, stale, err := c.memory.GetMetadataQuiet(key); err == nil {
		if recordMetrics {
			c.recordCacheHit()
		}
		return meta, stale, nil
	} else if !errors.Is(err, cache.ErrCacheEntryNotFound) {
		if recordMetrics {
			c.recordCacheError()
		}
		return nil, false, err
	}

	meta, stale, err = c.file.GetMetadataQuiet(key)
	if err != nil {
		if recordMetrics {
			if errors.Is(err, cache.ErrCacheEntryNotFound) {
				c.recordCacheMiss()
			} else {
				c.recordCacheError()
			}
		}
		return nil, false, err
	}

	if recordMetrics {
		c.recordCacheHit()
	}
	return meta, stale, nil
}

func (c *Cache[MetadataT]) GetMetadata(key cache.CacheKey) (meta *cache.EntryMetadata[MetadataT], stale bool, err error) {
	return c.getMetadata(key, true)
}

func (c *Cache[MetadataT]) GetMetadataQuiet(key cache.CacheKey) (meta *cache.EntryMetadata[MetadataT], stale bool, err error) {
	return c.getMetadata(key, false)
}

func (c *Cache[MetadataT]) UpdateMetadata(key cache.CacheKey, modifier func(*cache.EntryMetadata[MetadataT])) error {
	if err := c.memory.UpdateMetadataQuiet(key, modifier); err == nil {
		c.recordCacheHit()
		return nil
	} else if !errors.Is(err, cache.ErrCacheEntryNotFound) {
		c.recordCacheError()
		return err
	}

	if err := c.file.UpdateMetadataQuiet(key, modifier); err != nil {
		c.recordCacheError()
		return err
	}

	c.recordCacheHit()
	return nil
}

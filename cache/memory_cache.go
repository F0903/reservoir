package cache

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"reservoir/metrics"
	"reservoir/utils/atomics"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/mem"
)

var ErrCacheMemoryExceeded = fmt.Errorf("cache memory exceeded")

const INIT_BUFFER_SIZE = 1024 * 1

type memoryReadSeekCloser struct {
	*bytes.Reader
}

func (m *memoryReadSeekCloser) Close() error {
	return nil
}

type memoryInternalEntry[MetadataT any] struct {
	data []byte
	meta *EntryMetadata[MetadataT]
}

type MemoryCache[MetadataT any] struct {
	entries   map[CacheKey]*memoryInternalEntry[MetadataT]
	mu        sync.RWMutex
	locks     []sync.RWMutex
	memoryCap int64
	byteSize  atomics.Int64

	janitor *cacheJanitor[MetadataT]
}

func NewMemoryCache[MetadataT any](memoryBudgetPercent int, cleanupInterval time.Duration, shardCount int, ctx context.Context) *MemoryCache[MetadataT] {
	sysMem, err := mem.VirtualMemory()
	if err != nil {
		panic(fmt.Sprintf("failed to get system memory info: %v", err))
	}
	memoryCap := int64(sysMem.Total) * int64(memoryBudgetPercent) / 100

	c := &MemoryCache[MetadataT]{
		entries:   make(map[CacheKey]*memoryInternalEntry[MetadataT]),
		locks:     make([]sync.RWMutex, shardCount),
		memoryCap: memoryCap,
		byteSize:  atomics.NewInt64(0),
	}
	c.janitor = newCacheJanitor(cleanupInterval, cacheFunctions[MetadataT]{
		cacheIterator: func(yield func(key CacheKey, metadata *EntryMetadata[MetadataT]) bool) {
			c.mu.RLock()
			snapshot := maps.Clone(c.entries)
			c.mu.RUnlock()

			for key, entry := range snapshot {
				if !yield(key, entry.meta) {
					break
				}
			}
		},
		removeEntry: func(key CacheKey) error {
			return c.deleteInternal(key)
		},
		getCacheSize: func() int64 {
			return c.byteSize.Get()
		},
		getCacheLen: func() int {
			c.mu.RLock()
			defer c.mu.RUnlock()
			return len(c.entries)
		},
		getLock: func(key CacheKey) *sync.RWMutex {
			return getLock(c.locks, key)
		},
	})
	c.janitor.start(ctx)

	return c
}

func (c *MemoryCache[MetadataT]) Destroy() {
	c.janitor.stop()
}

func (c *MemoryCache[MetadataT]) Get(key CacheKey) (*Entry[MetadataT], error) {
	lock := getLock(c.locks, key)
	lock.Lock() // Using full lock to update LastAccess safely
	defer lock.Unlock()

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		metrics.Global.Cache.CacheMisses.Increment()
		return nil, ErrCacheEntryNotFound
	}

	stale := false
	if entry.meta.Expires.Before(time.Now()) {
		stale = true
	}

	entry.meta.LastAccess = time.Now()
	metrics.Global.Cache.CacheHits.Increment()

	return &Entry[MetadataT]{
		Data:     &memoryReadSeekCloser{bytes.NewReader(entry.data)},
		Metadata: entry.meta,
		Stale:    stale,
	}, nil
}

func (c *MemoryCache[MetadataT]) Cache(key CacheKey, data io.Reader, expires time.Time, metadata MetadataT) (*Entry[MetadataT], error) {
	lock := getLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	if c.byteSize.Get() >= c.memoryCap {
		metrics.Global.Cache.CacheErrors.Increment()
		return nil, ErrCacheMemoryExceeded
	}

	buf := bytes.NewBuffer(make([]byte, 0, INIT_BUFFER_SIZE))
	count, err := buf.ReadFrom(data)
	if err != nil {
		metrics.Global.Cache.CacheErrors.Increment()
		return nil, err
	}

	dataBytes := buf.Bytes()
	now := time.Now()
	meta := &EntryMetadata[MetadataT]{
		TimeWritten: now,
		LastAccess:  now,
		Expires:     expires,
		Size:        count,
		Object:      metadata,
	}
	internalEntry := &memoryInternalEntry[MetadataT]{
		data: dataBytes,
		meta: meta,
	}

	c.mu.Lock()
	c.entries[key] = internalEntry
	c.mu.Unlock()

	incrementCacheEntries()
	addCacheSize(&c.byteSize, int64(count))

	return &Entry[MetadataT]{
		Data:     &memoryReadSeekCloser{bytes.NewReader(dataBytes)},
		Metadata: meta,
	}, nil
}

func (c *MemoryCache[MetadataT]) Delete(key CacheKey) error {
	lock := getLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	return c.deleteInternal(key)
}

func (c *MemoryCache[MetadataT]) deleteInternal(key CacheKey) error {
	c.mu.Lock()
	entry, ok := c.entries[key]
	if !ok {
		c.mu.Unlock()
		return ErrCacheEntryNotFound
	}
	delete(c.entries, key)
	c.mu.Unlock()

	decrementCacheEntries()
	decrementCacheSize(&c.byteSize, entry.meta.Size)

	return nil
}

func (c *MemoryCache[MetadataT]) UpdateMetadata(key CacheKey, modifier func(*EntryMetadata[MetadataT])) error {
	lock := getLock(c.locks, key)
	lock.Lock()
	defer lock.Unlock()

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		metrics.Global.Cache.CacheErrors.Increment()
		return ErrCacheEntryNotFound
	}

	modifier(entry.meta)
	entry.meta.LastAccess = time.Now()

	metrics.Global.Cache.CacheHits.Increment()
	return nil
}

func (c *MemoryCache[MetadataT]) GetMetadata(key CacheKey) (meta *EntryMetadata[MetadataT], stale bool, err error) {
	lock := getLock(c.locks, key)
	lock.Lock() // Using full lock to update LastAccess safely
	defer lock.Unlock()

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		metrics.Global.Cache.CacheMisses.Increment()
		return nil, false, ErrCacheEntryNotFound
	}

	stale = false
	if entry.meta.Expires.Before(time.Now()) {
		stale = true
	}

	entry.meta.LastAccess = time.Now()
	metrics.Global.Cache.CacheHits.Increment()

	return entry.meta, stale, nil
}

package hybrid

import (
	"reservoir/cache"
	"time"
)

type placement int

const (
	placementMemoryShadowingFile placement = iota
	placementFileReplacingMemory
)

type writeResult[MetadataT any] struct {
	entry     *cache.Entry[MetadataT]
	placement placement
}

func (r writeResult[MetadataT]) entryForCaller() *cache.Entry[MetadataT] {
	return r.entry
}

func (r writeResult[MetadataT]) memoryShadowsFile() bool {
	return r.placement == placementMemoryShadowingFile
}

func (c *Cache[MetadataT]) enforceMaxCacheSize() {
	maxCacheSize := c.maxCacheSize.Get()
	if maxCacheSize <= 0 {
		return
	}

	memoryStats := c.memory.Stats()
	fileStats := c.file.Stats()
	if memoryStats.Bytes+fileStats.Bytes <= maxCacheSize {
		return
	}

	allowedFileBytes := maxCacheSize - memoryStats.Bytes
	if allowedFileBytes < 0 {
		allowedFileBytes = 0
	}
	c.file.EvictTo(allowedFileBytes)
}

func (c *Cache[MetadataT]) memoryLimit() int64 {
	return c.memory.CapacityBytes()
}

func (c *Cache[MetadataT]) makeMemoryRoom(key cache.CacheKey, size int64) bool {
	if c.memory.CanStore(key, size) {
		return true
	}

	demoteAfter := time.Duration(c.demoteAfter.Get())
	if demoteAfter > 0 {
		c.demoteEntriesOlderThan(time.Now().Add(-demoteAfter))
		if c.memory.CanStore(key, size) {
			return true
		}
	}

	limit := c.memoryLimit()
	if limit > 0 && c.memory.StoredBytes() >= limit {
		c.memory.EvictTo(limit)
		if c.memory.CanStore(key, size) {
			return true
		}
	}

	return false
}

func (c *Cache[MetadataT]) shouldStreamToFile(key cache.CacheKey, size int64) bool {
	if size <= 0 {
		return false
	}
	if size > c.memoryLimit() {
		return true
	}

	c.placementMu.Lock()
	defer c.placementMu.Unlock()
	return !c.makeMemoryRoom(key, size)
}

func (c *Cache[MetadataT]) memoryBufferLimitForUnknownSize(key cache.CacheKey) int64 {
	limit := c.memoryLimit()
	if limit <= 0 {
		return 0
	}

	c.placementMu.Lock()
	defer c.placementMu.Unlock()

	demoteAfter := time.Duration(c.demoteAfter.Get())
	if demoteAfter > 0 {
		c.demoteEntriesOlderThan(time.Now().Add(-demoteAfter))
	}

	if c.memory.StoredBytes() >= limit {
		c.memory.EvictTo(limit)
	}

	oldSize := c.memory.EntrySize(key)
	available := limit - (c.memory.StoredBytes() - oldSize)
	if available < 0 {
		return 0
	}
	return available
}

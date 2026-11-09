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

	allowedFileBytes := max(maxCacheSize-memoryStats.Bytes, 0)
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

// reserveMemory claims room to read size bytes into memory, counting other downloads already being read.
// On true the caller must release it with bufferedBytes.Sub; on false the data belongs on disk.
func (c *Cache[MetadataT]) reserveMemory(key cache.CacheKey, size int64) bool {
	if size <= 0 {
		return true
	}
	if size > c.memoryLimit() {
		return false
	}

	c.placementMu.Lock()
	defer c.placementMu.Unlock()

	if !c.makeMemoryRoom(key, size+c.bufferedBytes.Get()) {
		return false
	}
	c.bufferedBytes.Add(size)
	return true
}

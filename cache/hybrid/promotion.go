package hybrid

import (
	"errors"
	"io"
	"log/slog"
	"reservoir/cache"
)

func (c *Cache[MetadataT]) promoteToMemory(key cache.CacheKey, fileEntry *cache.Entry[MetadataT]) (*cache.Entry[MetadataT], bool) {
	if fileEntry == nil || fileEntry.Data == nil || fileEntry.Metadata == nil || fileEntry.Stale {
		return fileEntry, false
	}
	if fileEntry.Metadata.Size > c.memoryLimit() {
		return fileEntry, false
	}
	if _, err := fileEntry.Data.Seek(0, io.SeekStart); err != nil {
		slog.Debug("Skipping memory promotion because cached data could not be rewound", "key", key.Hex, "error", err)
		return fileEntry, false
	}
	data, err := io.ReadAll(fileEntry.Data)
	if err != nil {
		if _, seekErr := fileEntry.Data.Seek(0, io.SeekStart); seekErr != nil {
			slog.Warn("Failed to rewind file cache entry after memory promotion read failure", "key", key.Hex, "error", seekErr)
		}
		slog.Debug("Skipping memory promotion because cached data could not be read", "key", key.Hex, "error", err)
		return fileEntry, false
	}

	c.placementMu.Lock()
	defer c.placementMu.Unlock()

	promoted, err := c.cacheInMemoryShadowingFile(key, data, fileEntry.Metadata.Expires, fileEntry.Metadata.Object)
	if err != nil {
		if _, seekErr := fileEntry.Data.Seek(0, io.SeekStart); seekErr != nil {
			slog.Warn("Failed to rewind file cache entry after memory promotion failure", "key", key.Hex, "error", seekErr)
		}
		if !errors.Is(err, cache.ErrCacheMemoryExceeded) {
			slog.Debug("Failed to promote file cache entry into memory", "key", key.Hex, "error", err)
		}
		return fileEntry, false
	}

	_ = fileEntry.Data.Close()
	c.enforceMaxCacheSize()
	return promoted, true
}

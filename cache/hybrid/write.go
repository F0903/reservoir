package hybrid

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"reservoir/cache"
	memorycache "reservoir/cache/memory"
	"sync"
	"time"
)

type closeHookEntryData struct {
	cache.EntryData
	onClose   func()
	closeOnce sync.Once
}

func (d *closeHookEntryData) Close() error {
	err := d.EntryData.Close()
	d.closeOnce.Do(d.onClose)
	return err
}

func (c *Cache[MetadataT]) cacheInMemoryShadowingFile(key cache.CacheKey, data []byte, expires time.Time, metadata MetadataT) (*cache.Entry[MetadataT], error) {
	if !c.makeMemoryRoom(key, int64(len(data))) {
		return nil, cache.ErrCacheMemoryExceeded
	}

	entry, err := c.memory.CacheBytesStrictQuiet(key, data, expires, metadata)
	if errors.Is(err, cache.ErrCacheMemoryExceeded) {
		return nil, cache.ErrCacheMemoryExceeded
	}
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (c *Cache[MetadataT]) cacheBufferedInFileReplacingMemory(key cache.CacheKey, data []byte, expires time.Time, metadata MetadataT) (*cache.Entry[MetadataT], error) {
	fileEntry, err := c.file.Cache(key, bytes.NewReader(data), expires, metadata)
	if err != nil {
		return nil, err
	}
	if fileEntry.Data != nil {
		_ = fileEntry.Data.Close()
	}

	if err := c.memory.Delete(key); err != nil && !errors.Is(err, cache.ErrCacheEntryNotFound) {
		slog.Debug("Failed to remove memory cache entry after file placement", "key", key.Hex, "error", err)
	}
	c.enforceMaxCacheSize()
	return &cache.Entry[MetadataT]{
		Data:     memorycache.NewEntryData(data),
		Metadata: fileEntry.Metadata,
		Stale:    fileEntry.Stale,
	}, nil
}

func (c *Cache[MetadataT]) cacheStreamInFileReplacingMemory(key cache.CacheKey, data io.Reader, expires time.Time, metadata MetadataT) (*cache.Entry[MetadataT], error) {
	var fileEntry *cache.Entry[MetadataT]
	err := c.memory.DeleteAfter(key, func() error {
		var cacheErr error
		fileEntry, cacheErr = c.file.Cache(key, data, expires, metadata)
		return cacheErr
	})
	if err != nil && !errors.Is(err, cache.ErrCacheEntryNotFound) {
		return nil, err
	}

	if fileEntry == nil {
		return nil, cache.ErrCacheEntryNotFound
	}
	if fileEntry.Data != nil {
		fileEntry.Data = &closeHookEntryData{
			EntryData: fileEntry.Data,
			onClose:   c.enforceMaxCacheSize,
		}
	} else {
		c.enforceMaxCacheSize()
	}
	return fileEntry, nil
}

func (c *Cache[MetadataT]) cacheBuffered(key cache.CacheKey, data []byte, expires time.Time, metadata MetadataT) (*writeResult[MetadataT], error) {
	c.placementMu.Lock()
	defer c.placementMu.Unlock()

	entry, err := c.cacheInMemoryShadowingFile(key, data, expires, metadata)
	if err == nil {
		result := &writeResult[MetadataT]{
			entry:     entry,
			placement: placementMemoryShadowingFile,
		}
		if result.memoryShadowsFile() {
			c.enforceMaxCacheSize()
		}
		return result, nil
	}
	if !errors.Is(err, cache.ErrCacheMemoryExceeded) {
		c.recordCacheError()
		return nil, err
	}

	entry, err = c.cacheBufferedInFileReplacingMemory(key, data, expires, metadata)
	if err != nil {
		return nil, err
	}
	return &writeResult[MetadataT]{
		entry:     entry,
		placement: placementFileReplacingMemory,
	}, nil
}

func (c *Cache[MetadataT]) cacheUnknownSize(key cache.CacheKey, data io.Reader, expires time.Time, metadata MetadataT) (*cache.Entry[MetadataT], error) {
	dataBytes, spillReader, spilled, err := readUntilSpillThreshold(data, c.memoryBufferLimitForUnknownSize(key))
	if err != nil {
		c.recordCacheError()
		return nil, err
	}

	if spilled {
		entry, err := c.cacheStreamInFileReplacingMemory(key, spillReader, expires, metadata)
		if err != nil {
			c.recordCacheError()
			return nil, err
		}
		return entry, nil
	}

	result, err := c.cacheBuffered(key, dataBytes, expires, metadata)
	if err != nil {
		return nil, err
	}
	return result.entryForCaller(), nil
}

func (c *Cache[MetadataT]) cacheKnownSize(key cache.CacheKey, data io.Reader, size int64, expires time.Time, metadata MetadataT) (*cache.Entry[MetadataT], error) {
	if c.shouldStreamToFile(key, size) {
		entry, err := c.cacheStreamInFileReplacingMemory(key, data, expires, metadata)
		if err != nil {
			c.recordCacheError()
			return nil, err
		}
		return entry, nil
	}

	dataBytes, err := cache.ReadAllCacheBytes(data, c.memoryLimit())
	if err != nil {
		c.recordCacheError()
		return nil, err
	}

	result, err := c.cacheBuffered(key, dataBytes, expires, metadata)
	if err != nil {
		return nil, err
	}
	return result.entryForCaller(), nil
}

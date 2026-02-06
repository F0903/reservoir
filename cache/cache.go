package cache

import (
	"errors"
	"io"
	"time"
)

var (
	ErrCacheEntryNotFound = errors.New("cache entry not found")
)

type Cache[MetadataT any] interface {
	// Retrieves an entry from the cache by its input key.
	Get(key CacheKey) (*Entry[MetadataT], error)

	// Stores an entry in the cache with the specified input key, and returns the cached entry.
	Cache(key CacheKey, data io.Reader, expires time.Time, metadata MetadataT) (*Entry[MetadataT], error)

	// Removes an entry from the cache by its input key.
	Delete(key CacheKey) error

	// Retrieves the metadata of an entry in the cache.
	GetMetadata(key CacheKey) (meta *EntryMetadata[MetadataT], stale bool, err error)

	// Modifies the metadata of an entry in the cache.
	UpdateMetadata(key CacheKey, modifier func(*EntryMetadata[MetadataT])) error

	// Calls any cleanup operations that might be necessary. The cache must not be used after this method is called.
	Destroy()
}

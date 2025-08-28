package cache

import (
	"errors"
	"io"
	"time"
)

var (
	ErrCacheMiss          = errors.New("cache miss")
	ErrCacheEntryNotFound = errors.New("cache entry not found")
)

type Cache[ObjectData any] interface {
	// Retrieves an entry from the cache by its input key.
	Get(key CacheKey) (*Entry[ObjectData], error)

	// Stores an entry in the cache with the specified input key.
	Cache(key CacheKey, data io.Reader, expires time.Time, objectData ObjectData) error

	// Removes an entry from the cache by its input key.
	Delete(key CacheKey) error

	// Retrieves the metadata of an entry in the cache.
	GetMetadata(key CacheKey) (meta EntryMetadata[ObjectData], stale bool, err error)

	// Modifies the metadata of an entry in the cache.
	UpdateMetadata(key CacheKey, modifier func(*EntryMetadata[ObjectData])) error
}

package cache

import (
	"errors"
	"io"
	"time"
)

var (
	ErrorCacheMiss = errors.New("cache miss")
)

type Entry[ObjectData any] struct {
	Data     io.ReadCloser
	Metadata EntryMetadata[ObjectData]
	Stale    bool // Indicates if the entry is stale, meaning it has expired but is still present in the cache.
}

type EntryMetadata[ObjectData any] struct {
	TimeWritten time.Time  `json:"time_written"`
	Expires     time.Time  `json:"expires"`
	Object      ObjectData `json:"object"`
}

type Cache[ObjectData any] interface {
	// Get retrieves an entry from the cache by its input key.
	Get(key *CacheKey) (*Entry[ObjectData], error)

	// Cache stores an entry in the cache with the specified input key.
	Cache(key *CacheKey, data io.Reader, expires time.Time, objectData ObjectData) (*Entry[ObjectData], error)

	// Delete removes an entry from the cache by its input key.
	Delete(key *CacheKey) error

	// UpdateMetadata modifies the metadata of an entry in the cache.
	UpdateMetadata(key *CacheKey, modifier func(*EntryMetadata[ObjectData])) error
}

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
}

type EntryMetadata[ObjectData any] struct {
	TimeWritten time.Time  `json:"time_written"`
	Expires     time.Time  `json:"expires"`
	Object      ObjectData `json:"object"`
}

type Cache[ObjectData any] interface {
	// Get retrieves an entry from the cache by its input key.
	Get(input string) (*Entry[ObjectData], error)

	// Cache stores an entry in the cache with the specified input key.
	Cache(input string, data io.Reader, expires time.Time, objectData ObjectData) error

	// Delete removes an entry from the cache by its input key.
	Delete(input string) error
}

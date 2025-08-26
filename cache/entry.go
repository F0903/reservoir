package cache

import (
	"io"
	"time"
)

type EntryMetadata[ObjectData any] struct {
	TimeWritten time.Time  `json:"time_written"`
	LastAccess  time.Time  `json:"last_access"`
	Expires     time.Time  `json:"expires"`
	FileSize    int64      `json:"file_size"`
	Object      ObjectData `json:"object"`
}

type EntryData interface {
	io.ReadSeekCloser
	io.ReaderAt
}

type Entry[ObjectData any] struct {
	Data     EntryData
	Metadata *EntryMetadata[ObjectData]
	Stale    bool // Indicates if the entry is stale, meaning it has expired but is still present in the cache.
}

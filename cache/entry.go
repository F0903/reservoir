package cache

import (
	"io"
	"time"
)

type EntryMetadata[MetadataT any] struct {
	TimeWritten time.Time `json:"time_written"`
	LastAccess  time.Time `json:"last_access"`
	Expires     time.Time `json:"expires"`
	Size        int64     `json:"file_size"`
	Object      MetadataT `json:"object"`
}

type EntryData interface {
	io.ReadSeekCloser
	io.ReaderAt
}

type Entry[MetadataT any] struct {
	Data     EntryData
	Metadata *EntryMetadata[MetadataT]
	Stale    bool // Indicates if the entry is stale, meaning it has expired but is still present in the cache.
}

package cache

import (
	"apt_cacher_go/utils/bytesize"
	"io"
	"time"
)

type EntryMetadata[ObjectData any] struct {
	TimeWritten time.Time         `json:"time_written"`
	LastAccess  time.Time         `json:"last_access"`
	Expires     time.Time         `json:"expires"`
	FileSize    bytesize.ByteSize `json:"file_size"`
	Object      ObjectData        `json:"object"`
}

type Entry[ObjectData any] struct {
	Data     io.ReadCloser
	Metadata *EntryMetadata[ObjectData]
	Stale    bool // Indicates if the entry is stale, meaning it has expired but is still present in the cache.
}

package memory

import (
	"bytes"
	"reservoir/cache"
	"reservoir/utils/atomics"
	"time"
)

type memoryReadSeekCloser struct {
	*bytes.Reader
}

func (m *memoryReadSeekCloser) Close() error {
	return nil
}

func NewEntryData(data []byte) cache.EntryData {
	return &memoryReadSeekCloser{bytes.NewReader(data)}
}

type memoryInternalEntry[MetadataT any] struct {
	data       []byte
	meta       *cache.EntryMetadata[MetadataT]
	lastAccess atomics.Time
}

func newMemoryInternalEntry[MetadataT any](data []byte, meta *cache.EntryMetadata[MetadataT]) *memoryInternalEntry[MetadataT] {
	return &memoryInternalEntry[MetadataT]{
		data:       data,
		meta:       meta,
		lastAccess: atomics.NewAtomicTime(meta.LastAccess),
	}
}

func (e *memoryInternalEntry[MetadataT]) touch(now time.Time) {
	e.lastAccess.Set(now)
}

func (e *memoryInternalEntry[MetadataT]) setLastAccess(now time.Time) {
	e.meta.LastAccess = now
	e.lastAccess.Set(now)
}

func (e *memoryInternalEntry[MetadataT]) metadataSnapshot() *cache.EntryMetadata[MetadataT] {
	meta := *e.meta
	meta.LastAccess = e.lastAccess.Get()
	return &meta
}

package cache

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/shirou/gopsutil/v4/mem"
)

var ErrCacheMemoryExceeded = fmt.Errorf("cache memory exceeded")

const INIT_BUFFER_SIZE = 1024 * 1

type memoryReadSeekCloser struct {
	*bytes.Reader
}

func (m *memoryReadSeekCloser) Close() error {
	return nil
}

type MemoryCache[MetadataT any] struct {
	data         map[CacheKey]*Entry[MetadataT]
	memoryCap    int64
	currentUsage int64
}

func NewMemoryCache[MetadataT any](memoryBudgetPercent int) MemoryCache[MetadataT] {
	sysMem, err := mem.VirtualMemory()
	if err != nil {
		panic(fmt.Sprintf("failed to get system memory info: %v", err))
	}
	memoryCap := int64(sysMem.Total) * int64(memoryBudgetPercent) / 100

	return MemoryCache[MetadataT]{
		data:         make(map[CacheKey]*Entry[MetadataT]),
		memoryCap:    memoryCap,
		currentUsage: 0,
	}
}

func (m *MemoryCache[MetadataT]) Get(key CacheKey) (*Entry[MetadataT], error) {
	data, ok := m.data[key]
	if !ok {
		return nil, ErrCacheEntryNotFound
	}
	return data, nil
}

func (m *MemoryCache[MetadataT]) Cache(key CacheKey, data io.Reader, expires time.Time, metadata MetadataT) (*Entry[MetadataT], error) {
	if m.currentUsage >= m.memoryCap {
		return nil, ErrCacheMemoryExceeded
	}

	buf := bytes.NewBuffer(make([]byte, 0, INIT_BUFFER_SIZE))
	count, err := buf.ReadFrom(data)
	if err != nil {
		return nil, err
	}

	memReader := &memoryReadSeekCloser{bytes.NewReader(buf.Bytes())}
	m.currentUsage += int64(count)

	now := time.Now()
	entry := &Entry[MetadataT]{
		Data: memReader,
		Metadata: &EntryMetadata[MetadataT]{
			TimeWritten: now,
			LastAccess:  now,
			Expires:     expires,
			Size:        int64(count),
			Object:      metadata,
		},
	}
	m.data[key] = entry
	return entry, nil
}

func (m *MemoryCache[MetadataT]) Destroy() {
}

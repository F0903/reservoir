package cache

import (
	"context"
	"io"
	"reservoir/utils"
	"time"
)

type HybridCache[MetadataT any] struct {
	fCache *FileCache[MetadataT]
	mCache *MemoryCache[MetadataT]
}

func NewHybridCache[MetadataT any](memoryBudgetPercent int, rootDir string, cleanupInterval time.Duration, shardCount int, ctx context.Context) *HybridCache[MetadataT] {
	return &HybridCache[MetadataT]{
		fCache: NewFileCache[MetadataT](rootDir, cleanupInterval, shardCount, ctx),
		mCache: NewMemoryCache[MetadataT](memoryBudgetPercent, cleanupInterval, shardCount, ctx),
	}
}

func (h *HybridCache[MetadataT]) Get(key CacheKey) (*Entry[MetadataT], error) {
	//TODO
	return nil, utils.ErrNotImplemented
}

func (h *HybridCache[MetadataT]) Cache(key CacheKey, data io.Reader, expires time.Time, metadata MetadataT) (*Entry[MetadataT], error) {
	//TODO
	return nil, utils.ErrNotImplemented
}

func (h *HybridCache[MetadataT]) Delete(key CacheKey) error {
	//TODO
	return utils.ErrNotImplemented
}

func (h *HybridCache[MetadataT]) GetMetadata(key CacheKey) (meta *EntryMetadata[MetadataT], stale bool, err error) {
	//TODO
	return nil, false, utils.ErrNotImplemented
}

func (h *HybridCache[MetadataT]) UpdateMetadata(key CacheKey, modifier func(*EntryMetadata[MetadataT])) error {
	//TODO
	return utils.ErrNotImplemented
}

func (h *HybridCache[MetadataT]) Destroy() {
}

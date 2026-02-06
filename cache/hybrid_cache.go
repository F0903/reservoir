package cache

import (
	"context"
	"time"
)

type HybridCache[MetadataT any] struct {
	fCache *FileCache[MetadataT]
	mCache *MemoryCache[MetadataT]
}

func NewHybridCache[MetadataT any](memoryBudgetPercent int32, rootDir string, cleanupInterval time.Duration, ctx context.Context) HybridCache[MetadataT] {
	return HybridCache[MetadataT]{
		fCache: NewFileCache[MetadataT](rootDir, cleanupInterval, ctx),
	}
}

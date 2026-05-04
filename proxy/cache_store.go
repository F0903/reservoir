package proxy

import (
	"context"
	"fmt"
	"reservoir/cache"
	filecache "reservoir/cache/file"
	"reservoir/cache/hybrid"
	memorycache "reservoir/cache/memory"
	"reservoir/config"
)

func newCacheStore(cfg *config.Config, ctx context.Context) (cache.Cache[cachedRequestInfo], error) {
	cleanupInterval := cfg.Cache.CleanupInterval.Read().Cast()
	maxCacheSize := cfg.Cache.MaxCacheSize.Read().Bytes()
	shardCount := cfg.Cache.LockShards.Read()

	switch cfg.Cache.Type.Read() {
	case config.CacheTypeFile:
		cacheDir := cfg.Cache.File.Dir.Read()
		return filecache.New[cachedRequestInfo](cfg, cacheDir, maxCacheSize, cleanupInterval, shardCount, ctx), nil
	case config.CacheTypeHybrid:
		cacheDir := cfg.Cache.File.Dir.Read()
		memoryBudget := cfg.Cache.Memory.MemoryBudgetPercent.Read()
		return hybrid.New[cachedRequestInfo](cfg, cacheDir, memoryBudget, maxCacheSize, cleanupInterval, shardCount, ctx), nil
	case config.CacheTypeMemory:
		memoryBudget := cfg.Cache.Memory.MemoryBudgetPercent.Read()
		return memorycache.New[cachedRequestInfo](cfg, memoryBudget, maxCacheSize, cleanupInterval, shardCount, ctx), nil
	default:
		return nil, fmt.Errorf("unsupported cache type: %v", cfg.Cache.Type.Read())
	}
}

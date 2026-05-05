package config

import (
	"fmt"
	"reservoir/utils/bytesize"
	"reservoir/utils/duration"
	"time"
)

type CacheType string

var (
	CacheTypeFile   CacheType = "file"
	CacheTypeHybrid CacheType = "hybrid"
	CacheTypeMemory CacheType = "memory"
)

type FileCacheConfig struct {
	Dir ConfigProp[string] `json:"dir"` // The directory used by the file backend and hybrid file tier.
}

type MemoryCacheConfig struct {
	MemoryBudgetPercent ConfigProp[int] `json:"memory_budget_percent"` // The percentage of total memory used by the memory backend and hybrid memory tier.
}

type HybridCacheConfig struct {
	DemoteAfter ConfigProp[duration.Duration] `json:"demote_after"` // How long a hybrid memory-tier entry can sit without access before it is demoted to the file tier.
}

type CacheConfig struct {
	MaxCacheSize    ConfigProp[bytesize.ByteSize] `json:"max_cache_size"`   // The maximum size of the cache across all tiers.
	Type            ConfigProp[CacheType]         `json:"type"`             // The type of cache to use. Supported values are "memory", "file", and "hybrid".
	CleanupInterval ConfigProp[duration.Duration] `json:"cleanup_interval"` // The interval at which expired cache entries are removed.
	LockShards      ConfigProp[int]               `json:"lock_shards"`      // The number of shards to use for per-key locking.
	File            FileCacheConfig               `json:"file"`
	Memory          MemoryCacheConfig             `json:"memory"`
	Hybrid          HybridCacheConfig             `json:"hybrid"`
}

func (c *CacheConfig) setRestartNeededProps() {
	c.Type.SetRequiresRestart()
	c.File.Dir.SetRequiresRestart()
	c.LockShards.SetRequiresRestart()
}

func (c *CacheConfig) verify() error {
	if c.MaxCacheSize.Read().Bytes() <= 0 {
		return fmt.Errorf("cache.max_cache_size must be greater than 0")
	}
	if c.CleanupInterval.Read().Cast() <= 0 {
		return fmt.Errorf("cache.cleanup_interval must be greater than 0")
	}
	if c.Memory.MemoryBudgetPercent.Read() < 0 || c.Memory.MemoryBudgetPercent.Read() > 100 {
		return fmt.Errorf("cache.memory.memory_budget_percent must be between 0 and 100")
	}
	if c.Hybrid.DemoteAfter.Read().Cast() <= 0 {
		return fmt.Errorf("cache.hybrid.demote_after must be greater than 0")
	}
	if c.File.Dir.Read() == "" {
		return fmt.Errorf("cache.file.dir cannot be empty")
	}
	cType := c.Type.Read()
	if cType != CacheTypeFile && cType != CacheTypeMemory && cType != CacheTypeHybrid {
		return fmt.Errorf("cache.type must be one of 'file', 'memory', or 'hybrid'")
	}
	return nil
}

func defaultCacheConfig() CacheConfig {
	return CacheConfig{
		MaxCacheSize:    NewConfigProp(bytesize.ParseUnchecked("10G")),
		Type:            NewConfigProp(CacheTypeHybrid),
		CleanupInterval: NewConfigProp(duration.Duration(5 * time.Minute)),
		LockShards:      NewConfigProp(1024),
		File: FileCacheConfig{
			Dir: NewConfigProp("var/cache/"),
		},
		Memory: MemoryCacheConfig{
			MemoryBudgetPercent: NewConfigProp(25),
		},
		Hybrid: HybridCacheConfig{
			DemoteAfter: NewConfigProp(duration.Duration(5 * time.Minute)),
		},
	}
}

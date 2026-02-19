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
	CacheTypeMemory CacheType = "memory"
)

type FileCacheConfig struct {
	Dir ConfigProp[string] `json:"dir"` // The directory where cached files will be stored.
}

type MemoryCacheConfig struct {
	MemoryBudgetPercent ConfigProp[int] `json:"memory_budget_percent"` // The percentage of total memory to use for the cache.
}

type CacheConfig struct {
	MaxCacheSize    ConfigProp[bytesize.ByteSize] `json:"max_cache_size"`   // The maximum size of the cache in bytes.
	Type            ConfigProp[CacheType]         `json:"type"`             // The type of cache to use. Supported values are "memory" and "file".
	CleanupInterval ConfigProp[duration.Duration] `json:"cleanup_interval"` // The interval at which the cache will be cleaned up.
	LockShards      ConfigProp[int]               `json:"lock_shards"`      // The number of shards to use for per-key locking.
	File            FileCacheConfig               `json:"file"`
	Memory          MemoryCacheConfig             `json:"memory"`
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
	if c.File.Dir.Read() == "" {
		return fmt.Errorf("cache.file.dir cannot be empty")
	}
	if c.Type.Read() != CacheTypeFile && c.Type.Read() != CacheTypeMemory {
		return fmt.Errorf("cache.type must be either 'file' or 'memory'")
	}
	return nil
}

func defaultCacheConfig() CacheConfig {
	return CacheConfig{
		MaxCacheSize:    NewConfigProp(bytesize.ParseUnchecked("10G")),
		Type:            NewConfigProp(CacheTypeMemory),
		CleanupInterval: NewConfigProp(duration.Duration(90 * time.Minute)),
		LockShards:      NewConfigProp(1024),
		File: FileCacheConfig{
			Dir: NewConfigProp("var/cache/"),
		},
		Memory: MemoryCacheConfig{
			MemoryBudgetPercent: NewConfigProp(75),
		},
	}
}

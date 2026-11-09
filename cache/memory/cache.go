package memory

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path"
	"path/filepath"
	"reservoir/cache"
	"reservoir/config"
	"reservoir/metrics"
	"reservoir/utils/atomics"
	"reservoir/utils/bytesize"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/mem"
)

const (
	cgroupMountPath    = "/sys/fs/cgroup"
	procSelfCgroupPath = "/proc/self/cgroup"
)

type Cache[MetadataT any] struct {
	entries       map[cache.CacheKey]*memoryInternalEntry[MetadataT]
	mu            sync.RWMutex
	locks         []sync.RWMutex
	memoryCap     atomics.Int64
	maxCacheSize  atomics.Int64
	byteSize      atomics.Int64
	reservedBytes atomics.Int64 // Downloads being read into memory, not yet counted in byteSize.

	janitor *cache.Janitor[MetadataT]
	subs    config.ConfigSubscriber
	metrics *metrics.Metrics
}

func New[MetadataT any](cfg *config.Config, m *metrics.Metrics, memoryBudgetPercent int, maxCacheSize int64, cleanupInterval time.Duration, shardCount int, ctx context.Context) *Cache[MetadataT] {
	return newWithAggregateMetrics[MetadataT](cfg, m, memoryBudgetPercent, maxCacheSize, cleanupInterval, shardCount, ctx, true)
}

func NewTier[MetadataT any](cfg *config.Config, m *metrics.Metrics, memoryBudgetPercent int, maxCacheSize int64, cleanupInterval time.Duration, shardCount int, ctx context.Context) *Cache[MetadataT] {
	return newWithAggregateMetrics[MetadataT](cfg, m, memoryBudgetPercent, maxCacheSize, cleanupInterval, shardCount, ctx, false)
}

func newWithAggregateMetrics[MetadataT any](cfg *config.Config, m *metrics.Metrics, memoryBudgetPercent int, maxCacheSize int64, cleanupInterval time.Duration, shardCount int, ctx context.Context, trackAggregateMetrics bool) *Cache[MetadataT] {
	sysMem, err := mem.VirtualMemory()
	if err != nil {
		panic(fmt.Sprintf("failed to get system memory info: %v", err))
	}
	m = metrics.Ensure(m)
	totalMemory := int64(sysMem.Total)
	if runtime.GOOS == "linux" { // cgroups are Linux-only; on Windows this path would resolve against the current drive.
		totalMemory = memoryTotal(sysMem.Total, cgroupMountPath, procSelfCgroupPath)
	}

	c := &Cache[MetadataT]{
		entries:       make(map[cache.CacheKey]*memoryInternalEntry[MetadataT]),
		locks:         make([]sync.RWMutex, shardCount),
		memoryCap:     atomics.NewInt64(totalMemory * int64(memoryBudgetPercent) / 100),
		maxCacheSize:  atomics.NewInt64(maxCacheSize),
		byteSize:      atomics.NewInt64(0),
		reservedBytes: atomics.NewInt64(0),
		metrics:       m,
	}

	c.subs.Add(cfg.Cache.MaxCacheSize.OnChange(func(newSize bytesize.ByteSize) {
		c.maxCacheSize.Set(newSize.Bytes())
	}))

	c.subs.Add(cfg.Cache.Memory.MemoryBudgetPercent.OnChange(func(newPercent int) {
		newCap := totalMemory * int64(newPercent) / 100
		c.memoryCap.Set(newCap)
		slog.Info("Memory budget changed", "new_percent", newPercent, "new_cap", bytesize.ByteSize(newCap))
	}))

	c.janitor = cache.NewJanitor(cfg, cleanupInterval, cache.JanitorFunctions[MetadataT]{
		Iterate: func(yield func(key cache.CacheKey, metadata *cache.EntryMetadata[MetadataT]) bool) {
			c.mu.RLock()
			snapshot := maps.Clone(c.entries)
			c.mu.RUnlock()

			for key := range snapshot {
				lock := cache.GetLock(c.locks, key)
				// Eviction runs during writes, so this goroutine may already hold the shard's write lock.
				if !lock.TryRLock() {
					continue
				}
				c.mu.RLock()
				entry, ok := c.entries[key]
				c.mu.RUnlock()

				var meta *cache.EntryMetadata[MetadataT]
				if ok {
					meta = entry.metadataSnapshot()
				}
				lock.RUnlock()

				if !ok {
					continue
				}
				if !yield(key, meta) {
					break
				}
			}
		},
		Remove: func(key cache.CacheKey) error {
			return c.deleteInternal(key)
		},
		Size: func() int64 {
			return c.byteSize.Get()
		},
		Len: func() int {
			c.mu.RLock()
			defer c.mu.RUnlock()
			return len(c.entries)
		},
		Lock: func(key cache.CacheKey) *sync.RWMutex {
			return cache.GetLock(c.locks, key)
		},
	}, trackAggregateMetrics, trackAggregateMetrics, m)
	c.janitor.Start(ctx)

	return c
}

// memoryTotal returns the RAM available to this process: the host total, or the lowest
// limit set on its cgroup or any parent, since a parent's limit applies to everything below it.
// shortcut: reads cgroup v2 only, also check memory/memory.limit_in_bytes if cgroup v1 hosts need support.
func memoryTotal(hostTotal uint64, cgroupMount string, procCgroupFile string) int64 {
	total := int64(hostTotal)
	for dir := ownCgroup(procCgroupFile); ; dir = path.Dir(dir) {
		if limit, ok := readCgroupLimit(filepath.Join(cgroupMount, filepath.FromSlash(dir), "memory.max")); ok {
			total = min(total, limit)
		}
		if dir == "/" {
			return total
		}
	}
}

// ownCgroup returns this process's cgroup v2 path, or the root if it can't be determined.
func ownCgroup(procCgroupFile string) string {
	raw, err := os.ReadFile(procCgroupFile)
	if err != nil {
		return "/"
	}
	for line := range strings.Lines(string(raw)) {
		if own, ok := strings.CutPrefix(strings.TrimSpace(line), "0::"); ok {
			return path.Clean("/" + own) // A cgroup outside our namespace shows as "/..", which must not leave the mount.
		}
	}
	return "/"
}

func readCgroupLimit(file string) (int64, bool) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return 0, false
	}
	limit, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	return limit, err == nil && limit > 0 // An unlimited cgroup contains the word "max".
}

func (c *Cache[MetadataT]) recordCacheHit() {
	c.metrics.Cache.CacheHits.Increment()
}

func (c *Cache[MetadataT]) recordCacheMiss() {
	c.metrics.Cache.CacheMisses.Increment()
}

func (c *Cache[MetadataT]) recordCacheError() {
	c.metrics.Cache.CacheErrors.Increment()
}

func (c *Cache[MetadataT]) Destroy() {
	c.janitor.Stop()
	c.subs.UnsubscribeAll()
}

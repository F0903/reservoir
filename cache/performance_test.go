package cache_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reservoir/cache"
	filecache "reservoir/cache/file"
	"reservoir/cache/hybrid"
	memorycache "reservoir/cache/memory"
	"reservoir/config"
	"reservoir/utils/duration"
	stdatomic "sync/atomic"
	"testing"
	"time"
)

const (
	benchmarkMaxCacheSize = 2 * 1024 * 1024 * 1024
	benchmarkShardCount   = 32
	benchmarkReadBufSize  = 32 * 1024
)

type benchMetadata struct {
	ID string
}

var benchmarkReadChecksum stdatomic.Uint64

type cacheBenchmarkBackend struct {
	name string
	new  func(b *testing.B, cfg *config.Config) cache.Cache[benchMetadata]
}

var cacheBenchmarkBackends = []cacheBenchmarkBackend{
	{
		name: "Memory",
		new: func(b *testing.B, cfg *config.Config) cache.Cache[benchMetadata] {
			b.Helper()
			c := memorycache.New[benchMetadata](cfg, 50, benchmarkMaxCacheSize, time.Hour, benchmarkShardCount, context.Background())
			b.Cleanup(c.Destroy)
			return c
		},
	},
	{
		name: "File",
		new: func(b *testing.B, cfg *config.Config) cache.Cache[benchMetadata] {
			b.Helper()
			c := filecache.New[benchMetadata](cfg, b.TempDir(), benchmarkMaxCacheSize, time.Hour, benchmarkShardCount, context.Background())
			b.Cleanup(c.Destroy)
			return c
		},
	},
	{
		name: "Hybrid",
		new: func(b *testing.B, cfg *config.Config) cache.Cache[benchMetadata] {
			b.Helper()
			c := hybrid.New[benchMetadata](cfg, b.TempDir(), 50, benchmarkMaxCacheSize, time.Hour, benchmarkShardCount, context.Background())
			b.Cleanup(c.Destroy)
			return c
		},
	},
}

var cacheBenchmarkSizes = []struct {
	name string
	size int
}{
	{name: "1KB", size: 1024},
	{name: "256KB", size: 256 * 1024},
	{name: "4MB", size: 4 * 1024 * 1024},
}

func newBenchmarkConfig() *config.Config {
	cfg := config.NewDefault()
	cfg.Cache.Hybrid.DemoteAfter.Overwrite(duration.Duration(time.Hour))
	return cfg
}

func silenceBenchmarkLogs(b *testing.B) {
	b.Helper()

	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	b.Cleanup(func() {
		slog.SetDefault(previous)
	})
}

func benchmarkWorkingSetEntries(size int) int {
	switch {
	case size >= 4*1024*1024:
		return 32
	case size >= 256*1024:
		return 64
	default:
		return 256
	}
}

func benchmarkData(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i)
	}
	return data
}

func benchmarkKeys(prefix string, count int) []cache.CacheKey {
	keys := make([]cache.CacheKey, count)
	for i := range keys {
		keys[i] = cache.FromString(fmt.Sprintf("%s-%d", prefix, i))
	}
	return keys
}

func closeBenchmarkEntry(b *testing.B, entry *cache.Entry[benchMetadata]) {
	b.Helper()

	if entry == nil || entry.Data == nil {
		return
	}
	if err := entry.Data.Close(); err != nil {
		b.Fatalf("failed to close benchmark cache entry: %v", err)
	}
}

func readBenchmarkEntry(b *testing.B, entry *cache.Entry[benchMetadata], scratch []byte) {
	b.Helper()
	defer closeBenchmarkEntry(b, entry)

	checksum := uint64(0)
	for {
		n, err := entry.Data.Read(scratch)
		for _, value := range scratch[:n] {
			checksum += uint64(value)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			b.Fatalf("failed to read benchmark cache entry: %v", err)
		}
	}
	benchmarkReadChecksum.Add(checksum)
}

func seedBenchmarkCache(b *testing.B, c cache.Cache[benchMetadata], keys []cache.CacheKey, data []byte) {
	b.Helper()

	expires := time.Now().Add(time.Hour)
	for i, key := range keys {
		entry, err := c.Cache(key, bytes.NewReader(data), expires, benchMetadata{ID: fmt.Sprintf("seed-%d", i)})
		if err != nil {
			b.Fatalf("failed to seed benchmark cache: %v", err)
		}
		closeBenchmarkEntry(b, entry)
	}
}

func benchmarkWriteSameKey(b *testing.B, c cache.Cache[benchMetadata], data []byte) {
	key := cache.FromString("benchmark-write-same-key")
	expires := time.Now().Add(time.Hour)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry, err := c.Cache(key, bytes.NewReader(data), expires, benchMetadata{ID: "write-same"})
		if err != nil {
			b.Fatal(err)
		}
		closeBenchmarkEntry(b, entry)
	}
}

func benchmarkWriteWorkingSet(b *testing.B, c cache.Cache[benchMetadata], keys []cache.CacheKey, data []byte) {
	expires := time.Now().Add(time.Hour)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := keys[i%len(keys)]
		entry, err := c.Cache(key, bytes.NewReader(data), expires, benchMetadata{ID: "write-working-set"})
		if err != nil {
			b.Fatal(err)
		}
		closeBenchmarkEntry(b, entry)
	}
}

func benchmarkHotReadWorkingSet(b *testing.B, c cache.Cache[benchMetadata], keys []cache.CacheKey, data []byte) {
	seedBenchmarkCache(b, c, keys, data)
	scratch := make([]byte, benchmarkReadBufSize)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entry, err := c.Get(keys[i%len(keys)])
		if err != nil {
			b.Fatal(err)
		}
		readBenchmarkEntry(b, entry, scratch)
	}
}

func benchmarkParallelHotReadSameKey(b *testing.B, c cache.Cache[benchMetadata], keys []cache.CacheKey, data []byte) {
	seedBenchmarkCache(b, c, keys[:1], data)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		scratch := make([]byte, benchmarkReadBufSize)
		key := keys[0]
		for pb.Next() {
			entry, err := c.Get(key)
			if err != nil {
				b.Fatal(err)
			}
			readBenchmarkEntry(b, entry, scratch)
		}
	})
}

func benchmarkMixedReadWrite(b *testing.B, c cache.Cache[benchMetadata], keys []cache.CacheKey, data []byte) {
	seedBenchmarkCache(b, c, keys, data)
	expires := time.Now().Add(time.Hour)
	scratch := make([]byte, benchmarkReadBufSize)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := keys[i%len(keys)]
		if i%10 == 0 {
			entry, err := c.Cache(key, bytes.NewReader(data), expires, benchMetadata{ID: "mixed-write"})
			if err != nil {
				b.Fatal(err)
			}
			closeBenchmarkEntry(b, entry)
			continue
		}

		entry, err := c.Get(key)
		if err != nil {
			b.Fatal(err)
		}
		readBenchmarkEntry(b, entry, scratch)
	}
}

func benchmarkMetadataUpdate(b *testing.B, c cache.Cache[benchMetadata], keys []cache.CacheKey, data []byte) {
	seedBenchmarkCache(b, c, keys, data)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := keys[i%len(keys)]
		if err := c.UpdateMetadata(key, func(meta *cache.EntryMetadata[benchMetadata]) {
			meta.Object.ID = "metadata-update"
			meta.Expires = time.Now().Add(time.Hour)
		}); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkDeleteAndReadd(b *testing.B, c cache.Cache[benchMetadata], keys []cache.CacheKey, data []byte) {
	expires := time.Now().Add(time.Hour)

	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := keys[i%len(keys)]
		entry, err := c.Cache(key, bytes.NewReader(data), expires, benchMetadata{ID: "delete-readd"})
		if err != nil {
			b.Fatal(err)
		}
		closeBenchmarkEntry(b, entry)

		if err := c.Delete(key); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkMiss(b *testing.B, c cache.Cache[benchMetadata]) {
	key := cache.FromString("benchmark-missing-key")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := c.Get(key)
		if !errors.Is(err, cache.ErrCacheEntryNotFound) {
			b.Fatalf("expected cache miss, got %v", err)
		}
	}
}

func BenchmarkCacheBackends(b *testing.B) {
	silenceBenchmarkLogs(b)

	scenarios := []struct {
		name string
		run  func(b *testing.B, c cache.Cache[benchMetadata], keys []cache.CacheKey, data []byte)
	}{
		{name: "WriteSameKey", run: func(b *testing.B, c cache.Cache[benchMetadata], _ []cache.CacheKey, data []byte) {
			benchmarkWriteSameKey(b, c, data)
		}},
		{name: "WriteWorkingSet", run: benchmarkWriteWorkingSet},
		{name: "HotReadWorkingSet", run: benchmarkHotReadWorkingSet},
		{name: "ParallelHotReadSameKey", run: benchmarkParallelHotReadSameKey},
		{name: "Mixed90Read10Write", run: benchmarkMixedReadWrite},
		{name: "MetadataUpdate", run: benchmarkMetadataUpdate},
		{name: "DeleteAndReadd", run: benchmarkDeleteAndReadd},
		{name: "Miss", run: func(b *testing.B, c cache.Cache[benchMetadata], _ []cache.CacheKey, _ []byte) {
			benchmarkMiss(b, c)
		}},
	}

	for _, backend := range cacheBenchmarkBackends {
		for _, size := range cacheBenchmarkSizes {
			data := benchmarkData(size.size)
			keys := benchmarkKeys(fmt.Sprintf("%s-%s", backend.name, size.name), benchmarkWorkingSetEntries(size.size))

			for _, scenario := range scenarios {
				b.Run(fmt.Sprintf("%s/%s/%s", backend.name, size.name, scenario.name), func(b *testing.B) {
					cfg := newBenchmarkConfig()
					c := backend.new(b, cfg)
					scenario.run(b, c, keys, data)
				})
			}
		}
	}
}

package hybrid

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reservoir/cache"
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

type benchmarkUnknownSizeReader struct {
	io.Reader
}

func benchmarkUnknownReader(data []byte) io.Reader {
	return benchmarkUnknownSizeReader{Reader: bytes.NewReader(data)}
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

func BenchmarkHybridCacheTiering(b *testing.B) {
	silenceBenchmarkLogs(b)

	size := 256 * 1024
	data := benchmarkData(size)
	key := cache.FromString("benchmark-hybrid-promote-key")

	b.Run("ColdFileHitPromote", func(b *testing.B) {
		cfg := newBenchmarkConfig()
		c := New[benchMetadata](cfg, b.TempDir(), 50, benchmarkMaxCacheSize, time.Hour, benchmarkShardCount, context.Background())
		b.Cleanup(c.Destroy)

		expires := time.Now().Add(time.Hour)
		entry, err := c.Cache(key, bytes.NewReader(data), expires, benchMetadata{ID: "promote"})
		if err != nil {
			b.Fatal(err)
		}
		closeBenchmarkEntry(b, entry)

		b.ReportAllocs()
		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		scratch := make([]byte, benchmarkReadBufSize)
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			c.memory.OverrideEntryLastAccessForTesting(key, time.Now().Add(-2*time.Hour))
			c.demoteIdleEntries()
			b.StartTimer()

			entry, err := c.Get(key)
			if err != nil {
				b.Fatal(err)
			}
			readBenchmarkEntry(b, entry, scratch)
		}
	})

	b.Run("MemoryPressureSpillToFile", func(b *testing.B) {
		cfg := newBenchmarkConfig()
		c := New[benchMetadata](cfg, b.TempDir(), 50, benchmarkMaxCacheSize, time.Hour, benchmarkShardCount, context.Background())
		b.Cleanup(c.Destroy)
		c.memory.OverrideMemoryCapForTesting(int64(size + size/2))

		residentKey := cache.FromString("benchmark-hybrid-resident-key")
		residentEntry, err := c.Cache(residentKey, bytes.NewReader(data), time.Now().Add(time.Hour), benchMetadata{ID: "resident"})
		if err != nil {
			b.Fatal(err)
		}
		closeBenchmarkEntry(b, residentEntry)

		spillKeys := benchmarkKeys("benchmark-hybrid-spill", benchmarkWorkingSetEntries(size))

		b.ReportAllocs()
		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			entry, err := c.Cache(spillKeys[i%len(spillKeys)], bytes.NewReader(data), time.Now().Add(time.Hour), benchMetadata{ID: "spill"})
			if err != nil {
				b.Fatal(err)
			}
			closeBenchmarkEntry(b, entry)
		}
	})

	b.Run("UnknownLengthMemoryPressureSpillToFile", func(b *testing.B) {
		cfg := newBenchmarkConfig()
		c := New[benchMetadata](cfg, b.TempDir(), 50, benchmarkMaxCacheSize, time.Hour, benchmarkShardCount, context.Background())
		b.Cleanup(c.Destroy)
		c.memory.OverrideMemoryCapForTesting(int64(size + size/2))

		residentKey := cache.FromString("benchmark-hybrid-unknown-resident-key")
		residentEntry, err := c.Cache(residentKey, bytes.NewReader(data), time.Now().Add(time.Hour), benchMetadata{ID: "unknown-resident"})
		if err != nil {
			b.Fatal(err)
		}
		closeBenchmarkEntry(b, residentEntry)

		spillKeys := benchmarkKeys("benchmark-hybrid-unknown-spill", benchmarkWorkingSetEntries(size))

		b.ReportAllocs()
		b.SetBytes(int64(len(data)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			entry, err := c.Cache(spillKeys[i%len(spillKeys)], benchmarkUnknownReader(data), time.Now().Add(time.Hour), benchMetadata{ID: "unknown-spill"})
			if err != nil {
				b.Fatal(err)
			}
			closeBenchmarkEntry(b, entry)
		}
	})
}

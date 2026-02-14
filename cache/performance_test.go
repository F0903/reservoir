package cache

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"
)

type benchMetadata struct {
	ID string
}

func benchmarkCache(b *testing.B, cache Cache[benchMetadata], size int64) {
	data := make([]byte, size)
	key := CacheKey{Hex: "perf-test-key"}
	expires := time.Now().Add(1 * time.Hour)

	b.Run("Write", func(b *testing.B) {
		for b.Loop() {
			reader := bytes.NewReader(data)
			_, err := cache.Cache(key, reader, expires, benchMetadata{ID: "test"})
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Read", func(b *testing.B) {
		for b.Loop() {
			entry, err := cache.Get(key)
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, entry.Data)
			entry.Data.Close()
		}
	})
}

func BenchmarkCacheComparison(b *testing.B) {
	sizes := []struct {
		name string
		size int64
	}{
		{"1KB", 1024},
		{"1MB", 1024 * 1024},
	}

	for _, s := range sizes {
		b.Run(fmt.Sprintf("MemoryCache/%s", s.name), func(b *testing.B) {
			c := NewMemoryCache[benchMetadata](50, 1024*1024*1024, 1*time.Hour, 32, context.Background())
			defer c.Destroy()
			benchmarkCache(b, c, s.size)
		})

		b.Run(fmt.Sprintf("FileCache/%s", s.name), func(b *testing.B) {
			tmpDir := b.TempDir()
			c := NewFileCache[benchMetadata](tmpDir, 1024*1024*1024, 1*time.Hour, 32, context.Background())
			defer c.Destroy()
			benchmarkCache(b, c, s.size)
		})
	}
}

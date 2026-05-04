package cache

import (
	"reservoir/config"
	"sync"
	"testing"
	"time"
)

type janitorTestMeta struct {
	ID string
}

type janitorTestBackend struct {
	mu      sync.RWMutex
	locks   []sync.RWMutex
	entries map[CacheKey]*EntryMetadata[janitorTestMeta]
	size    int64
}

func newJanitorTestBackend() *janitorTestBackend {
	return &janitorTestBackend{
		locks:   make([]sync.RWMutex, 16),
		entries: make(map[CacheKey]*EntryMetadata[janitorTestMeta]),
	}
}

func (b *janitorTestBackend) put(key CacheKey, size int64, expires time.Time, lastAccess time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if previous, ok := b.entries[key]; ok {
		b.size -= previous.Size
	}
	b.entries[key] = &EntryMetadata[janitorTestMeta]{
		TimeWritten: time.Now(),
		LastAccess:  lastAccess,
		Expires:     expires,
		Size:        size,
		Object:      janitorTestMeta{ID: key.Hex},
	}
	b.size += size
}

func (b *janitorTestBackend) has(key CacheKey) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	_, ok := b.entries[key]
	return ok
}

func (b *janitorTestBackend) cachedBytes() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return b.size
}

func (b *janitorTestBackend) len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.entries)
}

func (b *janitorTestBackend) remove(key CacheKey) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	meta, ok := b.entries[key]
	if !ok {
		return ErrCacheEntryNotFound
	}
	delete(b.entries, key)
	b.size -= meta.Size
	return nil
}

func (b *janitorTestBackend) janitorFunctions() JanitorFunctions[janitorTestMeta] {
	return JanitorFunctions[janitorTestMeta]{
		Iterate: func(yield func(CacheKey, *EntryMetadata[janitorTestMeta]) bool) {
			b.mu.RLock()
			snapshot := make(map[CacheKey]*EntryMetadata[janitorTestMeta], len(b.entries))
			for key, metadata := range b.entries {
				metadataSnapshot := *metadata
				snapshot[key] = &metadataSnapshot
			}
			b.mu.RUnlock()

			for key, metadata := range snapshot {
				if !yield(key, metadata) {
					break
				}
			}
		},
		Remove: b.remove,
		Size:   b.cachedBytes,
		Len:    b.len,
		Lock: func(key CacheKey) *sync.RWMutex {
			return GetLock(b.locks, key)
		},
	}
}

func newTestJanitor(t *testing.T, backend *janitorTestBackend) *Janitor[janitorTestMeta] {
	t.Helper()

	j := NewJanitor(config.NewDefault(), time.Hour, backend.janitorFunctions(), false)
	t.Cleanup(j.subs.UnsubscribeAll)
	return j
}

func TestJanitor_CleansExpiredEntries(t *testing.T) {
	now := time.Now()
	expiredKey := FromString("expired-key")
	freshKey := FromString("fresh-key")
	backend := newJanitorTestBackend()
	backend.put(expiredKey, 400, now.Add(-time.Second), now)
	backend.put(freshKey, 600, now.Add(time.Hour), now)

	newTestJanitor(t, backend).cleanExpiredEntries()

	if backend.has(expiredKey) {
		t.Fatal("expected expired entry to be removed")
	}
	if !backend.has(freshKey) {
		t.Fatal("expected fresh entry to remain")
	}
	if got := backend.cachedBytes(); got != 600 {
		t.Fatalf("expected cached bytes to be 600 after cleanup, got %d", got)
	}
}

func TestJanitor_EvictsByPriorityUntilUnderTarget(t *testing.T) {
	now := time.Now()
	oldKey := FromString("old-key")
	recentKey := FromString("recent-key")
	backend := newJanitorTestBackend()
	backend.put(oldKey, 600, now.Add(time.Hour), now.Add(-time.Hour))
	backend.put(recentKey, 600, now.Add(time.Hour), now)

	newTestJanitor(t, backend).Evict(1024)

	if backend.has(oldKey) {
		t.Fatal("expected oldest entry to be evicted first")
	}
	if !backend.has(recentKey) {
		t.Fatal("expected recent entry to remain")
	}
	if got := backend.cachedBytes(); got != 600 {
		t.Fatalf("expected cached bytes to be 600 after eviction, got %d", got)
	}
}

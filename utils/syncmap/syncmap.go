package syncmap

import (
	"iter"
	"sync"
)

type SyncMap[K comparable, V any] struct {
	ma map[K]V
	mu sync.RWMutex // protects locks map
}

func New[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{
		ma: make(map[K]V),
	}
}

func (s *SyncMap[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		s.mu.RLock()
		keys := make([]K, 0, len(s.ma))
		for k := range s.ma {
			keys = append(keys, k)
		}
		s.mu.RUnlock()

		for _, k := range keys {
			if !yield(k) {
				return
			}
		}
	}
}

func (sm *SyncMap[K, V]) Get(key K) (V, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	val, ok := sm.ma[key]
	return val, ok
}

func (sm *SyncMap[K, V]) Delete(key K) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.ma, key)
}

func (sm *SyncMap[K, V]) Set(key K, value V) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.ma[key] = value
}

func (sm *SyncMap[K, V]) Len() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return len(sm.ma)
}

func (sm *SyncMap[K, V]) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	clear(sm.ma)
}

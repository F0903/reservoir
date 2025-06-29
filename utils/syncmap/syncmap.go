package syncmap

import "sync"

type SyncMap[K comparable, V any] struct {
	ma map[K]V
	mu sync.RWMutex // protects locks map
}

func NewSyncMap[K comparable, V any]() *SyncMap[K, V] {
	return &SyncMap[K, V]{
		ma: make(map[K]V),
	}
}

func (sm *SyncMap[K, V]) Get(key K) (V, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	val, ok := sm.ma[key]
	return val, ok
}

func (sm *SyncMap[K, V]) GetOrSet(key K, value V) V {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	val, ok := sm.ma[key]
	if !ok {
		val = value
		sm.ma[key] = value
	}
	return val
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

package writesynced

import "sync"

type WriteLock[T any] struct {
	mu  *sync.RWMutex
	val *T
}

// Locks and returns a pointer to the value held by the WriteLock.
// IMPORTANT: It is the responsibility of the caller to call Unlock when done.
func (w *WriteLock[T]) Get() *T {
	w.mu.Lock()
	return w.val
}

func (w *WriteLock[T]) Copy() T {
	w.mu.RLock()
	copy := *w.val
	w.mu.RUnlock()
	return copy
}

func (w *WriteLock[T]) Read(reader func(*T)) {
	w.mu.RLock()
	reader(w.val)
	w.mu.RUnlock()
}

// Releases the write lock acquired by Get
// IMPORTANT: The caller must ensure that UnGet is called exactly once for each call to Get.
func (w *WriteLock[T]) UnGet() {
	w.mu.Unlock()
}

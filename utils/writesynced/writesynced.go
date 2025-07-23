package writesynced

import (
	"sync"
)

type WriteSynced[T any] struct {
	mu  sync.RWMutex
	val T
}

func New[T any](initialValue T) *WriteSynced[T] {
	return &WriteSynced[T]{val: initialValue}
}

func (w *WriteSynced[T]) Get() T {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.val
}

func (w *WriteSynced[T]) Update(f func(*T)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	f(&w.val)
}

func (w *WriteSynced[T]) UpdateAndGet(f func(*T)) T {
	w.mu.Lock()
	defer w.mu.Unlock()
	f(&w.val)
	return w.val
}

func (w *WriteSynced[T]) Set(val T) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.val = val
}

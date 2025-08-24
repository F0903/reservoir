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

func (w *WriteSynced[T]) Immutable() ReadLock[T] {
	return ReadLock[T]{mu: &w.mu, val: &w.val}
}

func (w *WriteSynced[T]) Mutable() WriteLock[T] {
	return WriteLock[T]{mu: &w.mu, val: &w.val}
}

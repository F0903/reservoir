package writesynced

import "sync"

type ReadLock[T any] struct {
	mu  *sync.RWMutex
	val *T
}

func (r *ReadLock[T]) Copy() T {
	r.mu.RLock()
	copy := *r.val
	r.mu.RUnlock()
	return copy
}

func (r *ReadLock[T]) Read(reader func(*T)) {
	r.mu.RLock()
	reader(r.val)
	r.mu.RUnlock()
}

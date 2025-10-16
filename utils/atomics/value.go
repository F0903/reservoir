package atomics

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
)

type Value[T any] struct {
	v atomic.Value
}

func NewValue[T any](initial T) Value[T] {
	a := Value[T]{}
	a.v.Store(initial)
	return a
}

func (a *Value[T]) Store(val T) {
	a.v.Store(val) // make sure T is a concrete type (no nil pointers unless you store nil explicitly)
}

func (a *Value[T]) Load() (val T, some bool) {
	if v := a.v.Load(); v != nil {
		return v.(T), true
	}
	var zero T
	return zero, false
}

func (a *Value[T]) Swap(new T) (old T, some bool) {
	if oldAny := a.v.Swap(new); oldAny != nil {
		return oldAny.(T), true
	}
	return old, false
}

func (a *Value[T]) CompareAndSwap(old, new T) bool {
	return a.v.CompareAndSwap(old, new)
}

func (a *Value[T]) String() string {
	if v, some := a.Load(); some {
		return fmt.Sprintf("%v", v)
	}
	return "<nil>"
}

func (a *Value[T]) UnmarshalJSON(data []byte) error {
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("failed to unmarshal Value: %v", err)
	}
	a.Store(value)
	return nil
}

func (a Value[T]) MarshalJSON() ([]byte, error) {
	if v, some := a.Load(); some {
		return json.Marshal(v)
	}
	return nil, nil
}

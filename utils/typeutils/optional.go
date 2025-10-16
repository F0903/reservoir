package typeutils

import (
	"encoding/json"
	"errors"
)

var (
	ErrorUnwrapNone = errors.New("attempted to unwrap a None value")
)

type Optional[T any] struct {
	value T
	some  bool
}

func Some[T any](value T) Optional[T] {
	return Optional[T]{value: value, some: true}
}

func None[T any]() Optional[T] {
	var zero T
	return Optional[T]{value: zero, some: false}
}

func (o Optional[T]) IsSome() bool {
	return o.some
}

func (o Optional[T]) IsNone() bool {
	return !o.some
}

func (o Optional[T]) Get() (val T, ok bool) {
	if !o.some {
		var zero T
		return zero, false
	}
	return o.value, true
}

func (o Optional[T]) Unwrap() (T, error) {
	if !o.some {
		return *new(T), ErrorUnwrapNone
	}
	return o.value, nil
}

func (o Optional[T]) UnwrapOr(defaultValue T) T {
	if !o.some {
		return defaultValue
	}
	return o.value
}

func (o Optional[T]) ForceUnwrap() T {
	if !o.some {
		panic(ErrorUnwrapNone)
	}
	return o.value
}

func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.some {
		return []byte("null"), nil
	}
	return json.Marshal(o.value)
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		o.some = false
		return nil
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	o.value = value
	o.some = true

	return nil
}

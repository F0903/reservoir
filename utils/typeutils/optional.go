package typeutils

import (
	"encoding/json"
	"errors"
)

var (
	ErrorUnwrapNone = errors.New("attempted to unwrap a None value")
)

type Optional[T any] struct {
	value *T
}

func Some[T any](value *T) Optional[T] {
	return Optional[T]{value: value}
}

func None[T any]() Optional[T] {
	return Optional[T]{value: nil}
}

func (o Optional[T]) IsSome() bool {
	return o.value != nil
}

func (o Optional[T]) IsNone() bool {
	return o.value == nil
}

func (o Optional[T]) Unwrap() (T, error) {
	if o.value == nil {
		return *new(T), ErrorUnwrapNone
	}
	return *o.value, nil
}

func (o Optional[T]) UnwrapOr(defaultValue T) T {
	if o.value == nil {
		return defaultValue
	}
	return *o.value
}

func (o Optional[T]) ForceUnwrap() T {
	if o.value == nil {
		panic(ErrorUnwrapNone)
	}
	return *o.value
}

func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if o.value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(o.value)
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		o.value = nil
		return nil
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.value = &value
	return nil
}

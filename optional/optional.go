package optional

import "errors"

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

func (o Optional[T]) Unwrap() (*T, error) {
	if o.value == nil {
		return nil, ErrorUnwrapNone
	}
	return o.value, nil
}

func (o Optional[T]) ForceUnwrap() *T {
	if o.value == nil {
		panic(ErrorUnwrapNone)
	}
	return o.value
}

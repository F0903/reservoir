package config

import (
	"encoding/json"
	"reservoir/utils/typeutils"
)

// A generic type that can be overwritten with a new value, but retains the original value.
// This is useful for configurations or settings that can be temporarily changed, but should persist as the original value.
type overwritable[T any] struct {
	value       T
	overwritten typeutils.Optional[T]
}

func NewOverwritable[T any](value T) overwritable[T] {
	return overwritable[T]{
		value:       value,
		overwritten: typeutils.None[T](),
	}
}

// Returns the overwritten value if it exists, otherwise returns the original value.
func (o overwritable[T]) Get() T {
	return o.overwritten.UnwrapOr(o.value)
}

// Returns the original value, regardless of any overwrites.
func (o *overwritable[T]) Original() T {
	return o.value
}

// Sets the value and clears any overwrite.
func (o *overwritable[T]) Set(value T) {
	o.value = value
	o.ClearOverwrite()
}

// Sets the value without clearing any overwrite.
func (o *overwritable[T]) SetNoClear(value T) {
	o.value = value
}

func (o *overwritable[T]) Overwrite(value T) {
	o.overwritten = typeutils.Some(value)
}

func (o *overwritable[T]) IsOverwritten() bool {
	return o.overwritten.IsSome()
}

// Applies the overwrite if it exists, otherwise does nothing.
func (o *overwritable[T]) ApplyOverwrite() {
	if !o.IsOverwritten() {
		return
	}

	o.value = o.overwritten.ForceUnwrap()
	o.ClearOverwrite()
}

func (o *overwritable[T]) ClearOverwrite() {
	o.overwritten = typeutils.None[T]()
}

func (o overwritable[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.value)
}

func (o *overwritable[T]) UnmarshalJSON(data []byte) error {
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	o.value = value
	o.ClearOverwrite()
	return nil
}

package overwritable

import (
	"encoding/json"
	"reservoir/utils/optional"
)

// A generic type that can be overwritten with a new value, but retains the original value.
// This is useful for configurations or settings that can be temporarily changed, but should persist as the original value.
type Overwritable[T any] struct {
	value       T
	overwritten optional.Optional[T]
}

func New[T any](value T) Overwritable[T] {
	return Overwritable[T]{
		value:       value,
		overwritten: optional.None[T](),
	}
}

// Returns the overwritten value if it exists, otherwise returns the original value.
func (o *Overwritable[T]) Get() T {
	return o.overwritten.UnwrapOr(o.value)
}

// Returns the original value, regardless of any overwrites.
func (o *Overwritable[T]) Original() T {
	return o.value
}

// Sets the value and clears any overwrite.
func (o *Overwritable[T]) Set(value T) {
	o.value = value
	o.ClearOverwrite()
}

func (o *Overwritable[T]) Overwrite(value T) {
	o.overwritten = optional.Some(&value)
}

func (o *Overwritable[T]) IsOverwritten() bool {
	return o.overwritten.IsSome()
}

// Applies the overwrite if it exists, otherwise does nothing.
func (o *Overwritable[T]) ApplyOverwrite() {
	if !o.IsOverwritten() {
		return
	}

	o.value = o.overwritten.ForceUnwrap()
	o.ClearOverwrite()
}

func (o *Overwritable[T]) ClearOverwrite() {
	o.overwritten = optional.None[T]()
}

func (o Overwritable[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.value)
}

func (o *Overwritable[T]) UnmarshalJSON(data []byte) error {
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	o.value = value
	o.ClearOverwrite()
	return nil
}

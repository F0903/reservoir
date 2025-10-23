package config

import (
	"encoding/json"
	"fmt"
	"reservoir/utils/typeutils"
)

type commitable[T any] struct {
	comittedValue T
	stagedValue   typeutils.Optional[T]
}

func NewCommitable[T any](value T) commitable[T] {
	return commitable[T]{comittedValue: value, stagedValue: typeutils.None[T]()}
}

func (c *commitable[T]) ref() *T {
	return &c.comittedValue
}

func (c commitable[T]) Value() T {
	return c.comittedValue
}

func (c *commitable[T]) Stage(value T) {
	c.stagedValue = typeutils.Some(value)
}

func (c *commitable[T]) Commit() {
	if val, ok := c.stagedValue.Get(); ok {
		c.comittedValue = val
		c.stagedValue = typeutils.None[T]()
	}
}

func (c *commitable[T]) Uncommit() {
	c.stagedValue = typeutils.None[T]()
}

func (c *commitable[T]) String() string {
	if val, ok := c.stagedValue.Get(); ok {
		return fmt.Sprintf("Staged: %v", val)
	}
	return fmt.Sprintf("Committed: %v", c.comittedValue)
}

func (c commitable[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.comittedValue)
}

func (c *commitable[T]) UnmarshalJSON(data []byte) error {
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	c.comittedValue = value
	return nil
}

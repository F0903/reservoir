package config

import (
	"encoding/json"
	"fmt"
	"reservoir/utils/atomics"
	"reservoir/utils/event"
	"reservoir/utils/typeutils"
)

type Commitable[T any] struct {
	comittedValue T
	stagedValue   typeutils.Optional[T]
}

func NewCommitable[T any](value T) Commitable[T] {
	return Commitable[T]{comittedValue: value, stagedValue: typeutils.None[T]()}
}

func (c Commitable[T]) ref() *T {
	return &c.comittedValue
}

func (c Commitable[T]) Value() T {
	return c.comittedValue
}

func (c *Commitable[T]) Stage(value T) {
	c.stagedValue = typeutils.Some(value)
}

func (c *Commitable[T]) Commit() {
	if val, ok := c.stagedValue.Get(); ok {
		c.comittedValue = val
		c.stagedValue = typeutils.None[T]()
	}
}

func (c *Commitable[T]) Uncommit() {
	c.stagedValue = typeutils.None[T]()
}

type ConfigProp[T comparable] struct {
	// We use Overwritable to allow overrides from the command line while still retaining and persisting the original value from the config file.
	value           atomics.Value[Commitable[typeutils.Overwritable[T]]]
	onChange        event.Event[T]
	requiresRestart bool
}

func NewConfigProp[T comparable](value T) ConfigProp[T] {
	return ConfigProp[T]{value: atomics.NewValue(NewCommitable(typeutils.NewOverwritable(value)))}
}

func (p *ConfigProp[T]) SetRequiresRestart() {
	p.requiresRestart = true
}

func (p ConfigProp[T]) Read() T {
	commit, _ := p.value.Load() // Will always be set
	return commit.ref().Get()
}

func (p *ConfigProp[T]) OnChange(fn event.EventFn[T]) event.Unsubscribe {
	return p.onChange.Subscribe(fn)
}

func (p *ConfigProp[T]) Overwrite(value T) {
	commit, _ := p.value.Load()
	commit.ref().Overwrite(value)
	p.value.Store(commit)

	p.onChange.Fire(value)
}

// Stages the new value, keeping the old. The change is not committed until CommitStaged is called.
func (p *ConfigProp[T]) Stage(newValue T) {
	commit, _ := p.value.Load()

	oldVal := commit.ref().Original()

	// Copy the old Overwritable to keep any command-line overwrites.
	overwritable := commit.Value()
	overwritable.SetNoClear(newValue)
	commit.Stage(overwritable)

	p.value.Store(commit)

	if p.requiresRestart && (oldVal != newValue) {
		setRestartNeeded()
	}

	p.onChange.Fire(newValue)
}

func (p *ConfigProp[T]) CommitStaged() {
	commit, _ := p.value.Load()
	commit.Commit()
	p.value.Store(commit)
}

func (p *ConfigProp[T]) String() string {
	return fmt.Sprintf("%v", p.value)
}

func (p ConfigProp[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.value)
}

func (p *ConfigProp[T]) UnmarshalJSON(data []byte) error {
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("failed to unmarshal ConfigProp: %w", err)
	}

	p.Stage(value)
	return nil
}

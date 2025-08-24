package config

import (
	"encoding/json"
	"fmt"
	"reservoir/utils/event"
	"reservoir/utils/overwritable"
)

type ConfigProp[T any] struct {
	value    overwritable.Overwritable[T] // We use Overwritable to allow temporary overrides while still retaining and persisting the original value.
	onChange event.Event[T]
}

func NewConfigProp[T any](value T) ConfigProp[T] {
	return ConfigProp[T]{value: overwritable.New(value)}
}

func (p ConfigProp[T]) Read() T {
	return p.value.Get()
}

func (p *ConfigProp[T]) Update(f func(*T)) {
	if f == nil {
		return
	}

	p.Update(f)
}

func (p *ConfigProp[T]) OnChange(fn event.EventFn[T]) event.Unsubscribe {
	return p.onChange.Subscribe(fn)
}

func (p *ConfigProp[T]) Overwrite(value T) {
	p.value.Overwrite(value)
	p.onChange.Fire(value)
}

func (p *ConfigProp[T]) Set(value T) {
	p.value.Set(value)
	p.onChange.Fire(value)
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

	p.Set(value)
	return nil
}

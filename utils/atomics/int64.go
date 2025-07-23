package atomics

import (
	"encoding/json"
	"sync/atomic"
)

type Int64 struct {
	value int64
}

func (m *Int64) Add(delta int64) {
	atomic.AddInt64(&m.value, delta)
}

func (m *Int64) Increment() {
	m.Add(1)
}

func (m *Int64) Decrement() {
	m.Add(-1)
}

func (m *Int64) Set(value int64) {
	atomic.StoreInt64(&m.value, value)
}

func (m *Int64) Get() int64 {
	return atomic.LoadInt64(&m.value)
}

func (m Int64) MarshalJSON() ([]byte, error) {
	value := atomic.LoadInt64(&m.value)
	return json.Marshal(value)
}

func (m *Int64) UnmarshalJSON(data []byte) error {
	var value int64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	atomic.StoreInt64(&m.value, value)
	return nil
}

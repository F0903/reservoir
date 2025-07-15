package metrics

import (
	"encoding/json"
	"sync/atomic"
)

type AtomicInt64 struct {
	metric int64
}

func (m *AtomicInt64) Add(delta int64) {
	atomic.AddInt64(&m.metric, delta)
}

func (m *AtomicInt64) Increment() {
	m.Add(1)
}

func (m *AtomicInt64) Decrement() {
	m.Add(-1)
}

func (m *AtomicInt64) Set(value int64) {
	atomic.StoreInt64(&m.metric, value)
}

func (m *AtomicInt64) Get() int64 {
	return atomic.LoadInt64(&m.metric)
}

func (m AtomicInt64) MarshalJSON() ([]byte, error) {
	value := atomic.LoadInt64(&m.metric)
	return json.Marshal(value)
}

func (m *AtomicInt64) UnmarshalJSON(data []byte) error {
	var value int64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	atomic.StoreInt64(&m.metric, value)
	return nil
}

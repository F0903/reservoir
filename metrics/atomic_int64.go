package metrics

import "sync/atomic"

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

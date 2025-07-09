package metrics

import (
	"sync/atomic"
	"time"
)

type AtomicTime struct {
	timeMicro int64
}

func NewAtomicTime(initial time.Time) AtomicTime {
	at := AtomicTime{
		timeMicro: initial.UnixMicro(),
	}
	at.Set(initial)
	return at
}

func (t *AtomicTime) Set(value time.Time) {
	atomic.StoreInt64(&t.timeMicro, value.UnixMicro())
}

func (t *AtomicTime) Get() time.Time {
	return time.UnixMicro(atomic.LoadInt64(&t.timeMicro))
}

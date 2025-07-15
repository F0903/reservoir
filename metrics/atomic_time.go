package metrics

import (
	"encoding/json"
	"sync/atomic"
	"time"
)

type AtomicTime struct {
	timeMicro int64
}

func NewAtomicTime(initial time.Time) AtomicTime {
	return AtomicTime{
		timeMicro: initial.UnixMicro(),
	}
}

func (t *AtomicTime) Set(value time.Time) {
	atomic.StoreInt64(&t.timeMicro, value.UnixMicro())
}

func (t *AtomicTime) Get() time.Time {
	return time.UnixMicro(atomic.LoadInt64(&t.timeMicro))
}

func (t AtomicTime) MarshalJSON() ([]byte, error) {
	timeValue := time.UnixMicro(atomic.LoadInt64(&t.timeMicro))
	return json.Marshal(timeValue.Format(time.RFC3339))
}

func (t *AtomicTime) UnmarshalJSON(data []byte) error {
	var timeStr string
	if err := json.Unmarshal(data, &timeStr); err != nil {
		return err
	}

	timeValue, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return err
	}

	atomic.StoreInt64(&t.timeMicro, timeValue.UnixMicro())
	return nil
}

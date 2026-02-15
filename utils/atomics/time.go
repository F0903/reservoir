package atomics

import (
	"encoding/json"
	"sync/atomic"
	"time"
)

type Time struct {
	timeMicro *atomic.Int64
}

func NewAtomicTime(initial time.Time) Time {
	val := &atomic.Int64{}
	val.Store(initial.UnixMicro())
	return Time{
		timeMicro: val,
	}
}

func (t *Time) Set(value time.Time) {
	if t.timeMicro == nil {
		t.timeMicro = &atomic.Int64{}
	}
	t.timeMicro.Store(value.UnixMicro())
}

func (t *Time) Get() time.Time {
	if t.timeMicro == nil {
		return time.Time{}
	}
	return time.UnixMicro(t.timeMicro.Load())
}

func (t Time) MarshalJSON() ([]byte, error) {
	if t.timeMicro == nil {
		return json.Marshal(time.Time{}.Format(time.RFC3339))
	}
	timeValue := time.UnixMicro(t.timeMicro.Load())
	timeStr := timeValue.Format(time.RFC3339)
	return json.Marshal(timeStr)
}

func (t *Time) UnmarshalJSON(data []byte) error {
	var timeStr string
	if err := json.Unmarshal(data, &timeStr); err != nil {
		return err
	}

	timeValue, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return err
	}

	t.timeMicro.Store(timeValue.UnixMicro())
	return nil
}

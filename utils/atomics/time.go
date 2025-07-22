package atomics

import (
	"encoding/json"
	"sync/atomic"
	"time"
)

type Time struct {
	timeMicro int64
}

func NewAtomicTime(initial time.Time) Time {
	return Time{
		timeMicro: initial.UnixMicro(),
	}
}

func (t *Time) Set(value time.Time) {
	atomic.StoreInt64(&t.timeMicro, value.UnixMicro())
}

func (t *Time) Get() time.Time {
	return time.UnixMicro(atomic.LoadInt64(&t.timeMicro))
}

func (t Time) MarshalJSON() ([]byte, error) {
	timeValue := time.UnixMicro(atomic.LoadInt64(&t.timeMicro))
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

	atomic.StoreInt64(&t.timeMicro, timeValue.UnixMicro())
	return nil
}

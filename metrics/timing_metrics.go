package metrics

import "time"

type timingMetrics struct {
	StartTime AtomicTime `json:"start_time"`
}

func NewTimingMetrics() timingMetrics {
	return timingMetrics{
		StartTime: NewAtomicTime(time.Now()),
	}
}

package metrics

import "time"

type timingMetrics struct {
	StartTime AtomicTime
}

func NewTimingMetrics() timingMetrics {
	return timingMetrics{
		StartTime: NewAtomicTime(time.Now()),
	}
}

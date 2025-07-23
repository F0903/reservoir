package metrics

import (
	"reservoir/utils/atomics"
	"time"
)

type timingMetrics struct {
	StartTime atomics.Time `json:"start_time"`
}

func NewTimingMetrics() timingMetrics {
	return timingMetrics{
		StartTime: atomics.NewAtomicTime(time.Now()),
	}
}

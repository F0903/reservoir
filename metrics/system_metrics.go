package metrics

import (
	"reservoir/utils/atomics"
	"time"
)

type systemMetrics struct {
	StartTime atomics.Time `json:"start_time"`
}

func NewSystemMetrics() systemMetrics {
	return systemMetrics{
		StartTime: atomics.NewAtomicTime(time.Now()),
	}
}

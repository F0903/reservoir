package metrics

import (
	"reservoir/utils/atomics"
	"runtime"
	"time"
)

type systemMetrics struct {
	StartTime     atomics.Time   `json:"start_time"`
	MemAlloc      atomics.Uint64 `json:"mem_alloc_bytes"`
	MemTotalAlloc atomics.Uint64 `json:"mem_total_alloc_bytes"`
	MemSys        atomics.Uint64 `json:"mem_sys_bytes"`
	NumGoroutines atomics.Int64  `json:"num_goroutines"`
}

func NewSystemMetrics() systemMetrics {
	return systemMetrics{
		StartTime:     atomics.NewAtomicTime(time.Now()),
		MemAlloc:      atomics.NewUint64(0),
		MemTotalAlloc: atomics.NewUint64(0),
		MemSys:        atomics.NewUint64(0),
		NumGoroutines: atomics.NewInt64(0),
	}
}

// Collect gathers the latest runtime memory and goroutine statistics.
// This method is intended to be called periodically.
func (s *systemMetrics) Collect() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	s.MemAlloc.Set(m.Alloc)
	s.MemTotalAlloc.Set(m.TotalAlloc)
	s.MemSys.Set(m.Sys)
	s.NumGoroutines.Set(int64(runtime.NumGoroutine()))
}

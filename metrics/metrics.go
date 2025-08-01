package metrics

var Global Metrics = NewMetrics()

type Metrics struct {
	Cache    cacheMetrics   `json:"cache"`
	Requests requestMetrics `json:"requests"`
	Timing   timingMetrics  `json:"timing"`
}

func NewMetrics() Metrics {
	// Since Go always zero-initializes structs, we can just return a new instance with the StartTime set to now.
	return Metrics{
		Cache:    NewCacheMetrics(),
		Requests: NewRequestMetrics(),
		Timing:   NewTimingMetrics(),
	}
}

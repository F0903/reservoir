package metrics

var Global *Metrics = NewMetrics()

type Metrics struct {
	Cache    cacheMetrics   `json:"cache"`
	Requests requestMetrics `json:"requests"`
	System   systemMetrics  `json:"system"`
}

func NewMetrics() *Metrics {
	return &Metrics{
		Cache:    NewCacheMetrics(),
		Requests: NewRequestMetrics(),
		System:   NewSystemMetrics(),
	}
}

// Run collectors for metrics that are not event-driven, such as system stats.
func (m *Metrics) RunCollectors() {
	m.System.Collect()
}

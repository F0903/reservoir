package metrics

type requestMetrics struct {
	HTTPProxyRequests  AtomicInt64 `json:"http_proxy_requests"`
	HTTPSProxyRequests AtomicInt64 `json:"https_proxy_requests"`
	BytesServed        AtomicInt64 `json:"bytes_served"`
}

func NewRequestMetrics() requestMetrics {
	// Since Go always zero-initializes structs, we can just return a new "empty" instance.
	return requestMetrics{}
}

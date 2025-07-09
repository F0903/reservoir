package metrics

type requestMetrics struct {
	HTTPProxyequests   AtomicInt64
	HTTPSProxyRequests AtomicInt64
	BytesServed        AtomicInt64
}

func NewRequestMetrics() requestMetrics {
	// Since Go always zero-initializes structs, we can just return a new "empty" instance.
	return requestMetrics{}
}

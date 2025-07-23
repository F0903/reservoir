package metrics

import "apt_cacher_go/utils/atomics"

type requestMetrics struct {
	HTTPProxyRequests  atomics.Int64 `json:"http_proxy_requests"`
	HTTPSProxyRequests atomics.Int64 `json:"https_proxy_requests"`
	BytesServed        atomics.Int64 `json:"bytes_served"`
}

func NewRequestMetrics() requestMetrics {
	// Since Go always zero-initializes structs, we can just return a new "empty" instance.
	return requestMetrics{}
}

package metrics

import "reservoir/utils/atomics"

type requestMetrics struct {
	HTTPProxyRequests     atomics.Int64 `json:"http_proxy_requests"`
	HTTPSProxyRequests    atomics.Int64 `json:"https_proxy_requests"`
	BytesServed           atomics.Int64 `json:"bytes_served"`
	CoalescedRequests     atomics.Int64 `json:"coalesced_requests"`
	NonCoalescedRequests  atomics.Int64 `json:"non_coalesced_requests"`
	CoalescedCacheHits    atomics.Int64 `json:"coalesced_cache_hits"`
	CoalescedCacheMisses  atomics.Int64 `json:"coalesced_cache_misses"`
}

func NewRequestMetrics() requestMetrics {
	// Since Go always zero-initializes structs, we can just return a new "empty" instance.
	return requestMetrics{}
}

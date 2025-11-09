package metrics

import "reservoir/utils/atomics"

type requestMetrics struct {
	HTTPProxyRequests           atomics.Int64 `json:"http_proxy_requests"`
	HTTPSProxyRequests          atomics.Int64 `json:"https_proxy_requests"`
	BytesServed                 atomics.Int64 `json:"bytes_served"`
	CoalescedRequests           atomics.Int64 `json:"coalesced_requests"`
	NonCoalescedRequests        atomics.Int64 `json:"non_coalesced_requests"`
	CoalescedCacheHits          atomics.Int64 `json:"coalesced_cache_hits"`
	CoalescedCacheRevalidations atomics.Int64 `json:"coalesced_cache_revalidations"`
	CoalescedCacheMisses        atomics.Int64 `json:"coalesced_cache_misses"`
}

func NewRequestMetrics() requestMetrics {
	return requestMetrics{
		HTTPProxyRequests:           atomics.NewInt64(0),
		HTTPSProxyRequests:          atomics.NewInt64(0),
		BytesServed:                 atomics.NewInt64(0),
		CoalescedRequests:           atomics.NewInt64(0),
		NonCoalescedRequests:        atomics.NewInt64(0),
		CoalescedCacheHits:          atomics.NewInt64(0),
		CoalescedCacheRevalidations: atomics.NewInt64(0),
		CoalescedCacheMisses:        atomics.NewInt64(0),
	}
}

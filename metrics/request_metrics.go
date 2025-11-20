package metrics

import "reservoir/utils/atomics"

type requestMetrics struct {
	HTTPProxyRequests           atomics.Int64 `json:"http_proxy_requests"`
	HTTPSProxyRequests          atomics.Int64 `json:"https_proxy_requests"`
	BytesServed                 atomics.Int64 `json:"bytes_served"`
	BytesFetched                atomics.Int64 `json:"bytes_fetched"`
	UpstreamRequests            atomics.Int64 `json:"upstream_requests"`
	ClientRequestLatency        atomics.Int64 `json:"client_request_latency"`   // ns
	UpstreamRequestLatency      atomics.Int64 `json:"upstream_request_latency"` // ns
	CoalescedRequests           atomics.Int64 `json:"coalesced_requests"`
	NonCoalescedRequests        atomics.Int64 `json:"non_coalesced_requests"`
	CoalescedCacheHits          atomics.Int64 `json:"coalesced_cache_hits"`
	CoalescedCacheRevalidations atomics.Int64 `json:"coalesced_cache_revalidations"`
	CoalescedCacheMisses        atomics.Int64 `json:"coalesced_cache_misses"`
	StatusOKResponses           atomics.Int64 `json:"status_ok_responses"`
	StatusClientErrorResponses  atomics.Int64 `json:"status_client_error_responses"`
	StatusServerErrorResponses  atomics.Int64 `json:"status_server_error_responses"`
}

func NewRequestMetrics() requestMetrics {
	return requestMetrics{
		HTTPProxyRequests:           atomics.NewInt64(0),
		HTTPSProxyRequests:          atomics.NewInt64(0),
		BytesServed:                 atomics.NewInt64(0),
		BytesFetched:                atomics.NewInt64(0),
		UpstreamRequests:            atomics.NewInt64(0),
		ClientRequestLatency:        atomics.NewInt64(0),
		UpstreamRequestLatency:      atomics.NewInt64(0),
		CoalescedRequests:           atomics.NewInt64(0),
		NonCoalescedRequests:        atomics.NewInt64(0),
		CoalescedCacheHits:          atomics.NewInt64(0),
		CoalescedCacheRevalidations: atomics.NewInt64(0),
		CoalescedCacheMisses:        atomics.NewInt64(0),
		StatusOKResponses:           atomics.NewInt64(0),
		StatusClientErrorResponses:  atomics.NewInt64(0),
		StatusServerErrorResponses:  atomics.NewInt64(0),
	}
}

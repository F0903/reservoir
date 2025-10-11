import { apiGet, type FetchFn } from "$lib/api/api-methods";

export type RequestMetrics = {
    http_proxy_requests: number;
    https_proxy_requests: number;
    bytes_served: number;
    coalesced_requests: number;
    non_coalesced_requests: number;
    coalesced_cache_hits: number;
    coalesced_cache_misses: number;
};

export async function getRequestMetrics(
    fetchFn: FetchFn = fetch,
): Promise<Readonly<RequestMetrics>> {
    return apiGet<RequestMetrics>("/metrics/requests", fetchFn);
}

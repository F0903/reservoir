import { apiGet, type FetchFn } from "$lib/api/api-methods";

export type RequestMetrics = {
    http_proxy_requests: number;
    https_proxy_requests: number;
    bytes_served: number;
    bytes_fetched: number;
    coalesced_requests: number;
    non_coalesced_requests: number;
    coalesced_cache_hits: number;
    coalesced_cache_revalidations: number;
    coalesced_cache_misses: number;
    status_ok_responses: number;
    status_client_error_responses: number;
    status_server_error_responses: number;
};

export async function getRequestMetrics(
    fetchFn: FetchFn = fetch,
): Promise<Readonly<RequestMetrics>> {
    return apiGet<RequestMetrics>("/metrics/requests", fetchFn);
}

import { apiGet, type FetchFn } from "$lib/api/api-methods";

export type RequestMetrics = {
    http_proxy_requests: number;
    https_proxy_requests: number;
    bytes_served: number;
};

export async function getRequestMetrics(
    fetchFn: FetchFn = fetch,
): Promise<Readonly<RequestMetrics>> {
    return apiGet<RequestMetrics>("/metrics/requests", fetchFn);
}

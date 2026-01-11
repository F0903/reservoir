import { apiGet, type FetchFn } from "$lib/api/api-helpers";
import type { CacheMetrics } from "./cache-metrics";
import type { RequestMetrics } from "./request-metrics";
import type { SystemMetrics } from "./system-metrics";

export type Metrics = {
    cache: CacheMetrics;
    requests: RequestMetrics;
    system: SystemMetrics;
};

export async function getAllMetrics(fetchFn: FetchFn = fetch): Promise<Readonly<Metrics>> {
    return apiGet("/metrics", fetchFn);
}

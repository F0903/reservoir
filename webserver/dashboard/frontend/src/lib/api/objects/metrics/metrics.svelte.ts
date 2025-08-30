import { apiGet, type APIObjectConstructor, type FetchFn } from "$lib/api/api-object";
import { CacheMetrics } from "./cache-metrics.svelte";
import { RequestMetrics } from "./request-metrics.svelte";
import { SystemMetrics } from "./system-metrics.svelte";

export class Metrics {
    [key: string]: unknown;

    cache: CacheMetrics = new CacheMetrics({});
    requests: RequestMetrics = new RequestMetrics({});
    system: SystemMetrics = new SystemMetrics({});

    constructor(json: Record<string, unknown>) {
        this.updateFrom(json);
    }

    updateFrom = (json: Record<string, unknown>) => {
        if (json.cache) this.cache.updateFrom(json.cache as Record<string, unknown>);
        if (json.requests) this.requests.updateFrom(json.requests as Record<string, unknown>);
        if (json.system) this.system.updateFrom(json.system as Record<string, unknown>);
    };
}

export async function getAllMetrics<C extends APIObjectConstructor<T>, T>(
    type: C = CacheMetrics as C,
    fetchFn: FetchFn = fetch,
): Promise<T> {
    return apiGet("/metrics", type, fetchFn);
}

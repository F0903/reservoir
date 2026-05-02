import { apiGet, type FetchFn } from "$lib/api/api-helpers";

export type CacheStorageMetrics = {
    type: "memory" | "file";
    entries: number;
    bytes: number;
    max_bytes: number;
    memory_cap_bytes?: number;
};

export type CacheMetrics = {
    cache_hits: number;
    cache_misses: number;
    cache_errors: number;
    cache_request_hits: number;
    cache_request_misses: number;
    cache_request_revalidations: number;
    cache_request_stales: number;
    cache_entries: number;
    bytes_cached: number;
    cleanup_runs: number;
    bytes_cleaned: number;
    cache_evictions: number;
    cache_hit_latency: number;
    cache_miss_latency: number;
    storage: CacheStorageMetrics;
};

export async function getCacheMetrics(fetchFn: FetchFn = fetch): Promise<Readonly<CacheMetrics>> {
    return apiGet<CacheMetrics>("/metrics/cache", fetchFn);
}

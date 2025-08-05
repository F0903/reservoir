import { readPropOrDefault } from "$lib/utils/values";
import { apiGet, type FetchFn } from "../../api-object";

export class CacheMetrics {
    readonly cacheHits: number;
    readonly cacheMisses: number;
    readonly cacheErrors: number;
    readonly cacheEntries: number;
    readonly bytesCached: number;
    readonly cleanupRuns: number;
    readonly bytesCleaned: number;
    readonly cacheEvictions: number;

    constructor(json: Record<string, unknown>) {
        this.cacheHits = readPropOrDefault("cache_hits", json, 0);
        this.cacheMisses = readPropOrDefault("cache_misses", json, 0);
        this.cacheErrors = readPropOrDefault("cache_errors", json, 0);
        this.cacheEntries = readPropOrDefault("cache_entries", json, 0);
        this.bytesCached = readPropOrDefault("bytes_cached", json, 0);
        this.cleanupRuns = readPropOrDefault("cleanup_runs", json, 0);
        this.bytesCleaned = readPropOrDefault("bytes_cleaned", json, 0);
        this.cacheEvictions = readPropOrDefault("cache_evictions", json, 0);
    }
}

export async function getCacheMetrics(fetchFn: FetchFn = fetch): Promise<CacheMetrics> {
    return apiGet("/metrics/cache", CacheMetrics, fetchFn);
}

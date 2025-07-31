import { readJsonPropOrDefault, type JSONResponse } from "$lib/utils/json";
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

    constructor(json: JSONResponse) {
        this.cacheHits = readJsonPropOrDefault("cache_hits", json, 0);
        this.cacheMisses = readJsonPropOrDefault("cache_misses", json, 0);
        this.cacheErrors = readJsonPropOrDefault("cache_errors", json, 0);
        this.cacheEntries = readJsonPropOrDefault("cache_entries", json, 0);
        this.bytesCached = readJsonPropOrDefault("bytes_cached", json, 0);
        this.cleanupRuns = readJsonPropOrDefault("cleanup_runs", json, 0);
        this.bytesCleaned = readJsonPropOrDefault("bytes_cleaned", json, 0);
        this.cacheEvictions = readJsonPropOrDefault("cache_evictions", json, 0);
    }
}

export async function getCacheMetrics(fetchFn: FetchFn = fetch): Promise<CacheMetrics> {
    return apiGet("/metrics/cache", CacheMetrics, fetchFn);
}

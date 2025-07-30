import { apiGet, type JSONResponse } from "../../api-object";

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
        this.cacheHits = json["cache_hits"] as number;
        this.cacheMisses = json["cache_misses"] as number;
        this.cacheErrors = json["cache_errors"] as number;
        this.cacheEntries = json["cache_entries"] as number;
        this.bytesCached = json["bytes_cached"] as number;
        this.cleanupRuns = json["cleanup_runs"] as number;
        this.bytesCleaned = json["bytes_cleaned"] as number;
        this.cacheEvictions = json["cache_evictions"] as number;
    }
}

export async function getCacheMetrics(): Promise<CacheMetrics> {
    return apiGet("/metrics/cache", CacheMetrics);
}

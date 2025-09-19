import {
    apiGet,
    type FetchFn,
    type APIObjectConstructor,
    APIJsonObject,
} from "$lib/api/api-object";
import { setPropIfChanged } from "$lib/utils/objects/props";

export class CacheMetrics {
    cacheHits: number = $state(0);
    cacheMisses: number = $state(0);
    cacheErrors: number = $state(0);
    cacheEntries: number = $state(0);
    bytesCached: number = $state(0);
    cleanupRuns: number = $state(0);
    bytesCleaned: number = $state(0);
    cacheEvictions: number = $state(0);

    constructor(json: Record<string, unknown>) {
        this.updateFrom(json);
    }

    // prettier-ignore
    updateFrom = (json: Record<string, unknown>) => {
        setPropIfChanged("cache_hits",      json, this.cacheHits,       (value) => this.cacheHits = value as number);
        setPropIfChanged("cache_misses",    json, this.cacheMisses,     (value) => this.cacheMisses = value as number);
        setPropIfChanged("cache_errors",    json, this.cacheErrors,     (value) => this.cacheErrors = value as number);
        setPropIfChanged("cache_entries",   json, this.cacheEntries,    (value) => this.cacheEntries = value as number);
        setPropIfChanged("bytes_cached",    json, this.bytesCached,     (value) => this.bytesCached = value as number);
        setPropIfChanged("cleanup_runs",    json, this.cleanupRuns,     (value) => this.cleanupRuns = value as number);
        setPropIfChanged("bytes_cleaned",   json, this.bytesCleaned,    (value) => this.bytesCleaned = value as number);
        setPropIfChanged("cache_evictions", json, this.cacheEvictions,  (value) => this.cacheEvictions = value as number);
    }

    // Updates the cache metrics object by fetching from the API
    update = async () => {
        const data = await getCacheMetrics(APIJsonObject);
        this.updateFrom(data as Record<string, unknown>);
    };
}

export async function getCacheMetrics<C extends APIObjectConstructor<T>, T>(
    type: C = CacheMetrics as C,
    fetchFn: FetchFn = fetch,
): Promise<T> {
    return apiGet<T>("/metrics/cache", type, fetchFn);
}

import { apiGet, apiPost, type FetchFn } from "$lib/api/api-helpers";

export type CacheStatus = {
    type: "memory" | "file" | "hybrid";
    entries: number;
    bytes: number;
    max_bytes: number;
    memory_cap_bytes?: number;
};

export async function getCacheStatus(fetchFn: FetchFn = fetch): Promise<Readonly<CacheStatus>> {
    return apiGet<CacheStatus>("/cache/status", fetchFn);
}

export async function clearCache(fetchFn: FetchFn = fetch): Promise<void> {
    await apiPost<unknown>("/cache/clear", {}, fetchFn);
}

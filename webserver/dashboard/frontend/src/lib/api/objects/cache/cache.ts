import { apiPost, type FetchFn } from "$lib/api/api-helpers";

export async function clearCache(fetchFn: FetchFn = fetch): Promise<void> {
    await apiPost<unknown>("/cache/clear", {}, fetchFn);
}

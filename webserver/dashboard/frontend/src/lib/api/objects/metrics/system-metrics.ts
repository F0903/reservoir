import { apiGet, type FetchFn } from "$lib/api/api-methods";

export type SystemMetrics = {
    start_time: string;
    mem_alloc_bytes: number;
    mem_total_alloc_bytes: number;
    mem_sys_bytes: number;
    num_goroutines: number;
};

export async function getSystemMetrics(fetchFn: FetchFn = fetch): Promise<Readonly<SystemMetrics>> {
    return apiGet<SystemMetrics>("/metrics/system", fetchFn);
}

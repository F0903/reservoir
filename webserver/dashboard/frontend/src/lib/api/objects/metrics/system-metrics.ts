import { apiGet, type FetchFn } from "$lib/api/api-methods";

export type SystemMetrics = {
    start_time: string;
};

export async function getSystemMetrics(fetchFn: FetchFn = fetch): Promise<Readonly<SystemMetrics>> {
    return apiGet<SystemMetrics>("/metrics/system", fetchFn);
}

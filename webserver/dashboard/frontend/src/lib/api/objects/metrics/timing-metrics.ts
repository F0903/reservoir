import { readPropOrDefault } from "$lib/utils/values";
import { apiGet, type FetchFn } from "../../api-object";

export class TimingMetrics {
    readonly startTime: Date;

    constructor(json: Record<string, unknown>) {
        this.startTime = new Date(readPropOrDefault("start_time", json, ""));
    }
}

export async function getTimingMetrics(fetchFn: FetchFn = fetch): Promise<TimingMetrics> {
    return apiGet("/metrics/timing", TimingMetrics, fetchFn);
}

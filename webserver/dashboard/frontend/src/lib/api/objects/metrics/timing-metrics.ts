import { readPropOrDefault, type JSONResponse } from "$lib/utils/json";
import { apiGet, type FetchFn } from "../../api-object";

export class TimingMetrics {
    readonly startTime: Date;

    constructor(json: JSONResponse) {
        this.startTime = new Date(readPropOrDefault("start_time", json, ""));
    }
}

export async function getTimingMetrics(fetchFn: FetchFn = fetch): Promise<TimingMetrics> {
    return apiGet("/metrics/timing", TimingMetrics, fetchFn);
}

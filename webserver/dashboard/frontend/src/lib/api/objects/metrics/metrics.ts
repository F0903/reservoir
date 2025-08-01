import { readJsonPropOrDefault, type JSONResponse } from "$lib/utils/json";
import { apiGet, type FetchFn } from "../../api-object";
import { CacheMetrics } from "./cache-metrics";
import { RequestMetrics } from "./request-metrics";
import { TimingMetrics } from "./timing-metrics";

export class Metrics {
    readonly cache: CacheMetrics;
    readonly requests: RequestMetrics;
    readonly timing: TimingMetrics;

    constructor(json: JSONResponse) {
        this.cache = new CacheMetrics(readJsonPropOrDefault("cache", json, {}));
        this.requests = new RequestMetrics(readJsonPropOrDefault("requests", json, {}));
        this.timing = new TimingMetrics(readJsonPropOrDefault("timing", json, {}));
    }
}

export async function getAllMetrics(fetchFn: FetchFn = fetch): Promise<Metrics> {
    return apiGet("/metrics", Metrics, fetchFn);
}

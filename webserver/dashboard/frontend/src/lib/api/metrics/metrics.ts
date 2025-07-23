import { apiGet, type JSONResponse } from "../api-object";
import { CacheMetrics } from "./cache-metrics";
import { RequestMetrics } from "./request-metrics";
import { TimingMetrics } from "./timing-metrics";

class Metrics {
    readonly cache: CacheMetrics;
    readonly request: RequestMetrics;
    readonly timing: TimingMetrics;

    constructor(json: JSONResponse) {
        this.cache = new CacheMetrics(json["cache"] as JSONResponse);
        this.request = new RequestMetrics(json["request"] as JSONResponse);
        this.timing = new TimingMetrics(json["timing"] as JSONResponse);
    }
}

export async function getAllMetrics(): Promise<Metrics> {
    return apiGet("/metrics", Metrics);
}

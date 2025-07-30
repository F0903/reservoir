import { apiGet, type JSONResponse } from "../../api-object";

export class TimingMetrics {
    readonly startTime: Date;

    constructor(_json: JSONResponse) {
        this.startTime = new Date(_json["start_time"] as string);
    }
}

export async function getTimingMetrics(): Promise<TimingMetrics> {
    return apiGet("/metrics/timing", TimingMetrics);
}

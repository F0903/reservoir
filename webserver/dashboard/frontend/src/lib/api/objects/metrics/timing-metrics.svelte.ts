import { apiGet, type APIObjectConstructor, type FetchFn } from "$lib/api/api-object";
import { setPropIfChanged } from "$lib/utils/objects";
import { SvelteDate } from "svelte/reactivity";

export class TimingMetrics {
    startTime: SvelteDate = new SvelteDate();

    constructor(json: Record<string, unknown>) {
        this.updateFrom(json);
    }

    updateFrom(json: Record<string, unknown>) {
        setPropIfChanged("start_time", json, this.startTime, (value) =>
            this.startTime.setTime(Date.parse(value as string)),
        );
    }
}

export async function getTimingMetrics<C extends APIObjectConstructor<T>, T>(
    type: C = TimingMetrics as C,
    fetchFn: FetchFn = fetch,
): Promise<T> {
    return apiGet<T>("/metrics/timing", type, fetchFn);
}

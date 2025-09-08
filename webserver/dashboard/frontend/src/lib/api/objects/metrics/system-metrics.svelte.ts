import {
    apiGet,
    APIJsonObject,
    type APIObjectConstructor,
    type FetchFn,
} from "$lib/api/api-object";
import { log } from "$lib/utils/logger";
import { setPropIfChanged } from "$lib/utils/objects";
import { SvelteDate } from "svelte/reactivity";

export class SystemMetrics {
    startTime: SvelteDate = new SvelteDate();

    constructor(json: Record<string, unknown>) {
        this.updateFrom(json);
    }

    updateFrom(json: Record<string, unknown>) {
        setPropIfChanged(
            "start_time",
            json,
            this.startTime,
            (value) => {
                log.debug("Updating start_time:", value);
                this.startTime.setTime(Date.parse(value as string));
            },
            (_a, _b) => true, // Start time will never change during runtime
        );
    }

    // Updates the system metrics object by fetching from the API
    update = async () => {
        const data = await getSystemMetrics(APIJsonObject);
        this.updateFrom(data as Record<string, unknown>);
    };
}

export async function getSystemMetrics<C extends APIObjectConstructor<T>, T>(
    type: C = SystemMetrics as C,
    fetchFn: FetchFn = fetch,
): Promise<T> {
    return apiGet<T>("/metrics/system", type, fetchFn);
}

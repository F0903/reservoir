import type { FetchFn } from "$lib/api/api-object";
import { getAllMetrics, Metrics } from "$lib/api/objects/metrics/metrics";
import { log } from "$lib/utils/logger";

export class MetricsProvider {
    private metricsRefreshId: number;
    private readonly fetchFn: FetchFn;

    data: Metrics = $state(new Metrics({}));
    state: { initializing: boolean; error: unknown | null } = $state({
        initializing: true,
        error: null,
    });

    constructor(fetchFn: FetchFn = fetch) {
        this.fetchFn = fetchFn;
        this.metricsRefreshId = setInterval(() => this.refreshMetrics(), 10000);

        this.refreshMetrics();
    }

    private stopRefresh() {
        if (this.metricsRefreshId === null) return;

        clearInterval(this.metricsRefreshId);
    }

    private async refreshMetrics() {
        log.debug("Refreshing metrics...");
        this.state.error = null;

        try {
            this.data = await getAllMetrics(this.fetchFn);
            this.state.initializing = false;
        } catch (error) {
            this.state.error = error;
        }

        log.debug("Metrics refreshed:", this.data);
    }
}

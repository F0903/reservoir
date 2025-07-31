import type { FetchFn } from "$lib/api/api-object";
import { getAllMetrics, Metrics } from "$lib/api/objects/metrics/metrics";
import { doBrowser } from "$lib/utils/conditional";
import { log } from "$lib/utils/logger";

export class MetricsProvider {
    private metricsRefreshId: number;
    private readonly fetchFn: FetchFn;

    data: Metrics = $state(new Metrics({}));
    state: { initializing: boolean; error: unknown | null } = $state({
        initializing: true,
        error: null,
    });

    static async createAndRefresh(fetchFn: FetchFn = fetch): Promise<MetricsProvider> {
        const provider = new MetricsProvider(fetchFn);
        await provider.refreshMetrics();
        return provider;
    }

    private constructor(fetchFn: FetchFn = fetch) {
        this.fetchFn = fetchFn;
        this.metricsRefreshId = setInterval(() => this.refreshMetrics(), 10000);
    }

    private stopRefresh() {
        if (this.metricsRefreshId === null) return;

        clearInterval(this.metricsRefreshId);
    }

    async refreshMetrics() {
        log.debug("Refreshing metrics...");
        this.state.error = null;

        try {
            this.data = await getAllMetrics(this.fetchFn);
            this.state.initializing = false;
        } catch (error) {
            this.state.error = error;
        }

        log.debug("Metrics refreshed");
        doBrowser(() => {
            log.debug("Metrics data: ", this.data);
        });
    }
}

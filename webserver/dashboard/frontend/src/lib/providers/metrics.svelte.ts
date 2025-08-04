import type { FetchFn } from "$lib/api/api-object";
import { getAllMetrics, Metrics } from "$lib/api/objects/metrics/metrics";
import { doIfBrowser } from "$lib/utils/conditional";
import { log } from "$lib/utils/logger";

export class MetricsProvider {
    private refreshInterval: number;
    private metricsRefreshId: number | null = null;
    private readonly fetchFn: FetchFn;

    data: Metrics = $state(new Metrics({}));
    state: { initializing: boolean; error: unknown | null } = $state({
        initializing: true,
        error: null,
    });

    // Create a new MetricsProvider instance and immediately refresh metrics
    static async createAndRefresh(fetchFn: FetchFn = fetch): Promise<MetricsProvider> {
        const provider = new MetricsProvider(fetchFn);
        await provider.refreshMetrics();
        provider.startRefresh();
        return provider;
    }

    private constructor(fetchFn: FetchFn = fetch, refreshInterval: number = 10000) {
        this.fetchFn = fetchFn;
        this.refreshInterval = refreshInterval;
    }

    // Start the metrics refresh interval
    startRefresh() {
        if (this.metricsRefreshId !== null) return;

        log.debug("Starting metrics refresh interval");
        this.metricsRefreshId = setInterval(() => this.refreshMetrics(), this.refreshInterval);
    }

    // Stop the metrics refresh interval
    stopRefresh() {
        if (this.metricsRefreshId === null) return;

        log.debug("Stopping metrics refresh interval");
        clearInterval(this.metricsRefreshId);
        this.metricsRefreshId = null;
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
        doIfBrowser(() => {
            log.debug("Metrics data: ", this.data);
        });
    }
}

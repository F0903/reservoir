import { browser } from "$app/environment";
import { type FetchFn } from "$lib/api/api-methods";
import { getAllMetrics, type Metrics } from "$lib/api/objects/metrics/metrics";
import { log } from "$lib/utils/logger";
import type { SettingsProvider } from "./settings/settings-provider.svelte";
import type { LoadableState } from "$lib/utils/loadable";
import { patch } from "$lib/utils/objects/patch";

export class MetricsProvider {
    private settings: SettingsProvider;
    private metricsRefreshId: number | null = null;
    private readonly fetchFn: FetchFn;

    data: Metrics | null = $state(null);
    private state: LoadableState = $state({
        tag: "loading",
        errorMsg: null,
    });

    constructor(settings: SettingsProvider, fetchFn: FetchFn = fetch) {
        this.settings = settings;
        this.fetchFn = fetchFn;
    }

    getLoadableState(): LoadableState {
        return this.state;
    }

    // Start the metrics refresh interval
    startRefresh = () => {
        if (!browser) return; // Do not run this in SSR

        if (this.metricsRefreshId !== null) return;

        $effect(() => {
            const interval = this.settings.dashboardConfig.fields.updateInterval;
            log.debug("Updating metrics refresh interval to", interval);
            this.stopRefresh();
            this.metricsRefreshId = setInterval(() => this.refreshMetrics(), interval);
        });
    };

    // Stop the metrics refresh interval
    stopRefresh = () => {
        if (!browser) return; // Do not run this in SSR

        if (this.metricsRefreshId === null) return;

        log.debug("Stopping metrics refresh interval");
        clearInterval(this.metricsRefreshId);
        this.metricsRefreshId = null;
    };

    refreshMetrics = async () => {
        if (!browser) return; // Do not run this in SSR

        log.debug("Refreshing metrics...");

        try {
            const newData = await getAllMetrics(this.fetchFn);
            if (this.data === null) {
                this.data = newData;
            } else {
                patch(this.data, newData);
            }
            this.state = { tag: "ok", errorMsg: null };
        } catch (error) {
            this.state = { tag: "error", errorMsg: error as string };
        }

        log.debug("Metrics refreshed");
        log.debug("Metrics data: ", this.data);
    };
}

import { browser } from "$app/environment";
import { type FetchFn } from "$lib/api/api-helpers";
import { getAllMetrics, type Metrics } from "$lib/api/objects/metrics/metrics";
import { log } from "$lib/utils/logger";
import type { SettingsProvider } from "./settings/settings-provider.svelte";
import { patch } from "$lib/utils/patch";

export class MetricsProvider {
    private settings: SettingsProvider;
    private metricsRefreshId: number | null = null;
    private readonly fetchFn: FetchFn;

    data: Metrics | null = $state(null);
    error: string | null = null;

    constructor(settings: SettingsProvider, fetchFn: FetchFn = fetch) {
        this.settings = settings;
        this.fetchFn = fetchFn;
    }

    // Start the metrics refresh interval
    startRefresh = () => {
        if (!browser) return; // Do not run this in SSR

        if (this.metricsRefreshId !== null) return;

        $effect(() => {
            const interval = this.settings.dashboardSettings.fields.updateInterval;
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
            this.error = null;
            const newData = await getAllMetrics(this.fetchFn);
            if (this.data === null) {
                log.debug("Metrics data was null, replacing with new data");
                this.data = newData;
            } else {
                log.debug("Patching existing metrics data with new data");
                patch(this.data, newData);
            }
        } catch (error) {
            this.error = String(error);
        }

        log.debug("Metrics refreshed");
        log.debug("Metrics data: ", $state.snapshot(this.data));
    };
}

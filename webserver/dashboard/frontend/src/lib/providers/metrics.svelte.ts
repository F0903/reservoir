import { browser } from "$app/environment";
import { type FetchFn, RawAPIObject } from "$lib/api/api-object";
import { getAllMetrics, Metrics } from "$lib/api/objects/metrics/metrics.svelte";
import { log } from "$lib/utils/logger";
import { getContext } from "svelte";
import type { SettingsProvider } from "./settings.svelte";
import { get, type Unsubscriber } from "svelte/store";

export class MetricsProvider {
    private settings: SettingsProvider;
    private metricsRefreshId: number | null = null;
    private readonly fetchFn: FetchFn;
    private updateIntervalUnsub: Unsubscriber | null = null;

    data: Metrics = new Metrics({});
    state: { initializing: boolean; error: unknown | null } = $state({
        initializing: true,
        error: null,
    });

    // Create a new MetricsProvider instance and immediately refresh metrics
    static createAndRefresh(fetchFn: FetchFn = fetch): MetricsProvider {
        const settings = getContext("settings") as SettingsProvider;

        const provider = new MetricsProvider(fetchFn, settings);
        provider.startRefresh();
        provider.refreshMetrics(); // Do not wait for it to complete, just start it and move on
        return provider;
    }

    private constructor(fetchFn: FetchFn = fetch, settings: SettingsProvider) {
        this.fetchFn = fetchFn;
        this.settings = settings;
    }

    // Start the metrics refresh interval
    startRefresh = () => {
        if (!browser) return; // Do not run this in SSR

        if (this.metricsRefreshId !== null) return;

        log.debug("Starting metrics refresh interval");
        this.metricsRefreshId = setInterval(
            () => this.refreshMetrics(),
            get(this.settings.dashboardConfig.updateInterval),
        );

        this.updateIntervalUnsub = this.settings.dashboardConfig.updateInterval.subscribe(
            (interval) => {
                log.debug("Updating metrics refresh interval to", interval);
                this.stopRefresh();
                this.metricsRefreshId = setInterval(() => this.refreshMetrics(), interval);
            },
        );
    };

    // Stop the metrics refresh interval
    stopRefresh = () => {
        if (!browser) return; // Do not run this in SSR

        if (this.metricsRefreshId === null) return;

        log.debug("Stopping metrics refresh interval");
        clearInterval(this.metricsRefreshId);
        this.metricsRefreshId = null;

        if (this.updateIntervalUnsub) {
            this.updateIntervalUnsub();
            this.updateIntervalUnsub = null;
        }
    };

    refreshMetrics = async () => {
        if (!browser) return; // Do not run this in SSR

        log.debug("Refreshing metrics...");
        this.state.error = null;

        try {
            const newData = await getAllMetrics(RawAPIObject, this.fetchFn);
            this.data.updateFrom(newData as Record<string, unknown>);
            this.state.initializing = false;
        } catch (error) {
            this.state.error = error;
        }

        log.debug("Metrics refreshed");
        log.debug("Metrics data: ", this.data);
    };
}

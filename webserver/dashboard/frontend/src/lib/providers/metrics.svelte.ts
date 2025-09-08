import { browser } from "$app/environment";
import { type FetchFn, APIJsonObject } from "$lib/api/api-object";
import { getAllMetrics, Metrics } from "$lib/api/objects/metrics/metrics.svelte";
import { log } from "$lib/utils/logger";
import { getContext } from "svelte";
import type { SettingsProvider } from "./settings/settings-provider.svelte";
import type { LoadableState } from "$lib/utils/loadable";

export class MetricsProvider {
    private settings: SettingsProvider;
    private metricsRefreshId: number | null = null;
    private readonly fetchFn: FetchFn;

    readonly data: Metrics = new Metrics({});
    private state: LoadableState = $state({
        tag: "loading",
        errorMsg: null,
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

    getLoadableState(): LoadableState {
        return this.state;
    }

    // Start the metrics refresh interval
    startRefresh = () => {
        if (!browser) return; // Do not run this in SSR

        if (this.metricsRefreshId !== null) return;

        log.debug("Starting metrics refresh interval");
        this.metricsRefreshId = setInterval(
            () => this.refreshMetrics(),
            this.settings.dashboardConfig.fields.updateInterval,
        );

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
            const newData = await getAllMetrics(APIJsonObject, this.fetchFn);
            this.data.updateFrom(newData as Record<string, unknown>);
            this.state = { tag: "ok", errorMsg: null };
        } catch (error) {
            this.state = { tag: "error", errorMsg: error as string };
        }

        log.debug("Metrics refreshed");
        log.debug("Metrics data: ", this.data);
    };
}

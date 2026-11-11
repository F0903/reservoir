import { browser } from "$app/env";
import { type FetchFn } from "#lib/api/api-helpers.ts";
import { getAllMetrics, type Metrics } from "#lib/api/objects/metrics/metrics.ts";
import { log } from "#lib/utils/logger.ts";
import type { SettingsProvider } from "../settings/settings-provider.svelte";
import { patch } from "#lib/utils/patch.ts";
import { SvelteDate } from "svelte/reactivity";

export class MetricsProvider {
    private settings: SettingsProvider;
    private readonly fetchFn: FetchFn;
    private refreshing = false;
    private refreshPromise: Promise<void> | null = null;
    private abortController: AbortController | null = null;

    data = $state<Metrics | null>(null);
    error = $state<string | null>(null);
    loading = $state(false);
    lastUpdated: SvelteDate | null = null;

    constructor(settings: SettingsProvider, fetchFn: FetchFn = fetch) {
        this.settings = settings;
        this.fetchFn = fetchFn;
    }

    // Start the metrics refresh loop
    startRefresh = () => {
        if (!browser || this.refreshing) return;

        log.debug("Starting metrics refresh loop");
        this.refreshing = true;
        this.abortController = new AbortController();
        this.refreshLoop();
    };

    // Stop the metrics refresh loop
    stopRefresh = () => {
        if (!this.refreshing) return;

        log.debug("Stopping metrics refresh loop");
        this.refreshing = false;
        this.abortController?.abort();
        this.abortController = null;
    };

    private refreshLoop = async () => {
        while (this.refreshing) {
            const startTime = Date.now();
            await this.refreshMetrics();

            if (!this.refreshing) break;

            // Calculate remaining wait time to respect the interval accurately
            // The setting comes from a free-text field: empty or 0 would mean polling back to back.
            const interval = Math.max(
                500,
                Number(this.settings.dashboardSettings.fields.updateInterval) || 10000,
            );
            const elapsed = Date.now() - startTime;
            const waitTime = Math.max(0, interval - elapsed);

            log.debug(`Waiting ${waitTime}ms for next metrics refresh...`);

            try {
                // Wait for the interval or until aborted
                await new Promise((resolve, reject) => {
                    const timeout = setTimeout(resolve, waitTime);
                    this.abortController?.signal.addEventListener(
                        "abort",
                        () => {
                            clearTimeout(timeout);
                            reject(new Error("aborted"));
                        },
                        { once: true },
                    );
                });
            } catch {
                // If aborted, the loop will exit because this.refreshing is false
                break;
            }
        }
    };

    refreshMetrics = () => {
        if (!browser) return; // Do not run this in SSR
        if (this.refreshPromise) return this.refreshPromise;

        this.refreshPromise = this.runRefreshMetrics();
        return this.refreshPromise;
    };

    private runRefreshMetrics = async () => {
        log.debug("Refreshing metrics...");
        this.loading = true;

        try {
            const newData = await getAllMetrics(this.fetchFn);

            // Clear error if we successfully fetched data
            this.error = null;

            if (this.data === null) {
                log.debug("Metrics data was null, replacing with new data");
                this.data = newData;
            } else {
                log.debug("Patching existing metrics data with new data");
                patch(this.data, newData);
            }
            this.lastUpdated = new SvelteDate();
        } catch (err) {
            // Ignore abort errors
            if (err instanceof Error && (err.name === "AbortError" || err.message === "aborted")) {
                return;
            }
            log.error("Failed to refresh metrics:", err);
            this.error = err instanceof Error ? err.message : String(err);
        } finally {
            this.loading = false;
            this.refreshPromise = null;
        }

        log.debug("Metrics refreshed");
    };
}

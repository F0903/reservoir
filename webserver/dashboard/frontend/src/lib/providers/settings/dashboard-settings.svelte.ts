import { browser } from "$app/environment";
import { log } from "$lib/utils/logger";
import type { Settings } from "./settings-provider.svelte";

type Fields = {
    updateInterval: number;
};

// Manages browser stored settings for the dashboard
export class DashboardSettings implements Settings {
    fields: Fields = $state({
        updateInterval: 10000,
    });

    constructor() {
        this.reload();

        $effect(() => {
            if (this.fields.updateInterval) {
                this.save();
            }
        });
    }

    reload = () => {
        if (!browser) {
            // If SSR, just return
            return Promise.resolve();
        }

        let configJson = localStorage.getItem("dashboardConfig");
        log.debug("Reloading dashboard settings from localStorage...", configJson);
        if (!configJson) {
            log.debug("No dashboard settings found in localStorage, saving defaults...");
            this.save(); // Try to save defaults if nothing is present

            configJson = localStorage.getItem("dashboardConfig");
            if (!configJson) {
                return Promise.resolve();
            }
        }

        let savedData;
        try {
            savedData = JSON.parse(configJson);
        } catch (e) {
            log.error("Failed to parse dashboard settings from localStorage:", e);
            return Promise.resolve();
        }

        log.debug("Parsed dashboard settings from localStorage:", savedData);
        if (savedData.updateInterval == this.fields.updateInterval) {
            log.debug("No changes detected in dashboard settings.");
            return Promise.resolve();
        }

        this.fields.updateInterval = savedData.updateInterval;

        log.debug("Updated dashboard settings from localStorage:", this.fields);
        return Promise.resolve();
    };

    save = () => {
        if (!browser) return; // Do not run this in SSR

        localStorage.setItem("dashboardConfig", JSON.stringify(this.fields));
    };
}

import { browser } from "$app/environment";
import {
    defaultDashboardLayout,
    normalizeDashboardLayout,
    type DashboardWidgetLayout,
} from "$lib/dashboard/dashboard-layout";
import { log } from "$lib/utils/logger";
import type { Settings } from "./settings-provider.svelte";

type DashboardSettingsFields = {
    updateInterval: number;
    layout: DashboardWidgetLayout[];
};

// Manages browser stored settings for the dashboard
export class DashboardSettings implements Settings {
    fields: DashboardSettingsFields = $state({
        updateInterval: 10000,
        layout: defaultDashboardLayout(),
    });

    constructor() {
        if (browser) {
            this.loadFromLocalStorage();

            // Auto-save whenever fields change
            $effect.root(() => {
                $effect(() => {
                    this.save();
                });
            });
        }
    }

    private loadFromLocalStorage = () => {
        const configJson = localStorage.getItem("dashboardConfig");
        if (!configJson) return;

        try {
            const savedData = JSON.parse(configJson);
            if (typeof savedData !== "object" || savedData === null) {
                return;
            }

            if (typeof savedData.updateInterval === "number") {
                this.fields.updateInterval = savedData.updateInterval;
            }
            this.fields.layout = normalizeDashboardLayout(savedData.layout);
            log.debug("Loaded dashboard settings from localStorage:", $state.snapshot(this.fields));
        } catch (e) {
            log.error("Failed to parse dashboard settings from localStorage:", e);
        }
    };

    reload = async () => {
        this.loadFromLocalStorage();
        return Promise.resolve();
    };

    save = () => {
        if (!browser) return; // Do not run this in SSR

        log.debug("Saving dashboard settings to localStorage:", $state.snapshot(this.fields));
        localStorage.setItem("dashboardConfig", JSON.stringify(this.fields));
    };
}

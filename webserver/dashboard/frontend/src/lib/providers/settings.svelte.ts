import { browser } from "$app/environment";
import { writable, get, type Writable } from "svelte/store";

class DashboardConfig {
    updateInterval: Writable<number> = writable(10000);

    [key: string]: Writable<number> | (() => unknown);

    static loadOrCreate(): DashboardConfig {
        if (!browser) {
            // If SSR, return a default config
            return new DashboardConfig();
        }

        const configJson = localStorage.getItem("dashboardConfig");
        if (!configJson) {
            return new DashboardConfig();
        }

        const savedData = JSON.parse(configJson);
        const config = new DashboardConfig();

        for (const [key, value] of Object.entries(savedData)) {
            if (typeof value === "function") continue;

            const writable = config[key] as Writable<unknown>;
            writable.set(value);
        }

        return config;
    }

    save = () => {
        if (!browser) return; // Do not run this in SSR

        // Extract values from stores before serializing
        const configData: Record<string, unknown> = {};
        for (const [key, value] of Object.entries(this)) {
            if (typeof value === "function") continue;

            const rawValue = get(value);
            configData[key] = rawValue;
        }

        localStorage.setItem("dashboardConfig", JSON.stringify(configData));
    };
}

export class SettingsProvider {
    dashboardConfig: DashboardConfig = DashboardConfig.loadOrCreate();
}

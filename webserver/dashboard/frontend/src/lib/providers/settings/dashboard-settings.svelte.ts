import { browser } from "$app/environment";
import type { Settings } from "./settings-provider.svelte";
import { setPropIfChanged } from "$lib/utils/objects";

class Fields {
    updateInterval = $state(10000);
}

// Manages browser stored settings for the dashboard
export class DashboardSettings implements Settings {
    fields = new Fields();

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
        if (!configJson) {
            this.save(); // Try to save defaults if nothing is present

            configJson = localStorage.getItem("dashboardConfig");
            if (!configJson) {
                return Promise.resolve();
            }
        }

        const savedData = JSON.parse(configJson);
        setPropIfChanged(
            "updateInterval",
            savedData,
            this.fields.updateInterval,
            (value) => (this.fields.updateInterval = value as number),
        );

        return Promise.resolve();
    };

    save = () => {
        if (!browser) return; // Do not run this in SSR

        localStorage.setItem("dashboardConfig", JSON.stringify(this.fields));
    };
}

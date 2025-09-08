import { browser } from "$app/environment";
import { Config } from "$lib/api/objects/config/config.svelte";
import type { Settings } from "./settings-provider.svelte";

// Manages proxy settings from API.
export class ProxySettings implements Settings {
    fields: Config = new Config({});

    constructor() {
        this.reload();
    }

    reload = async (): Promise<void> => {
        if (!browser) {
            // If SSR, just return
            return;
        }

        await this.fields.update();
    };
}

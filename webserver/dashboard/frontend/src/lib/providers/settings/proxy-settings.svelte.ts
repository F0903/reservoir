import { browser } from "$app/environment";
import { getConfig, type Config } from "$lib/api/objects/config/config";
import { getRestartRequired } from "$lib/api/objects/config/restart-required";
import { log } from "$lib/utils/logger";
import { patch } from "$lib/utils/objects/patch";
import type { Settings } from "./settings-provider.svelte";

// Manages proxy settings from API.
export class ProxySettings implements Settings {
    fields: Config | null = $state(null);
    needsRestart = $state(false);

    constructor() {
        this.reload();
    }

    reload = async (): Promise<void> => {
        if (!browser) {
            // If SSR, just return
            return;
        }

        const newData = await getConfig();
        if (this.fields === null) {
            this.fields = newData;
        } else {
            patch(this.fields, newData);
        }

        this.needsRestart = (await getRestartRequired()).restart_required;
        log.debug("Needs restart:", this.needsRestart);
    };
}

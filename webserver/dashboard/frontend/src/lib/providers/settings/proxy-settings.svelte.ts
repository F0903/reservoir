import { browser } from "$app/environment";
import { Config } from "$lib/api/objects/config/config.svelte";
import { getRestartRequired } from "$lib/api/objects/config/restart-required.svelte";
import { log } from "$lib/utils/logger";
import type { Settings } from "./settings-provider.svelte";

// Manages proxy settings from API.
export class ProxySettings implements Settings {
    fields: Config = new Config({});
    needsRestart = $state(false);

    constructor() {
        this.reload();
    }

    reload = async (): Promise<void> => {
        if (!browser) {
            // If SSR, just return
            return;
        }

        await this.fields.update();
        this.needsRestart = (await getRestartRequired()).restart_required;
        log.debug("Needs restart:", this.needsRestart);
    };
}

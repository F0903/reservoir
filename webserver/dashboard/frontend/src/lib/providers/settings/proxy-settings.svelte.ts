import { browser } from "$app/environment";
import { getConfig, type Config } from "$lib/api/objects/config/config";
import { getRestartRequired } from "$lib/api/objects/config/restart-required";
import { log } from "$lib/utils/logger";
import { patch } from "$lib/utils/patch";
import type { Settings } from "./settings-provider.svelte";

// Gets proxy settings from API.
export class ProxySettings implements Settings {
    fields: Config = $state({} as Config);
    needsRestart = $state(false);

    reload = async (): Promise<void> => {
        if (!browser) {
            // If SSR, just return
            return;
        }

        log.debug("Reloading proxy settings from API...", $state.snapshot(this.fields));
        const newData = await getConfig();
        patch(this.fields, newData);
        log.debug("Updated proxy settings from API: (snapshot)", $state.snapshot(this.fields));

        this.needsRestart = (await getRestartRequired()).restart_required;
        log.debug("Needs restart:", this.needsRestart);
    };
}

import { MetricsProvider } from "$lib/providers/metrics.svelte";
import type { LayoutLoad } from "./$types";
export const load: LayoutLoad = async ({ fetch }) => {
    return {
        metrics: await MetricsProvider.createAndRefresh(fetch),
    };
};

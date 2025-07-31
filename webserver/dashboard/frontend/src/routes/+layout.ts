import { MetricsProvider } from "$lib/providers/metrics.svelte";
import type { LayoutLoad } from "./$types";
export const load: LayoutLoad = ({ fetch }) => {
    return {
        metrics: new MetricsProvider(fetch),
    };
};

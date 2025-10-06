<script lang="ts">
    import { formatTimeSinceDate } from "$lib/utils/dates";
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import PolledWidget from "./base/PolledWidget.svelte";
    import { getContext } from "svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import { SvelteDate } from "svelte/reactivity";

    let metrics = getContext("metrics") as MetricsProvider;

    let currentUptime: string = $state("N/A");

    function updateUptime() {
        if (!metrics) {
            return;
        }

        if (!metrics.data) {
            currentUptime = "N/A";
            return;
        }

        // There are much more efficient ways of doing this, but cant be bothered
        const startTime = metrics.data.system.start_time;
        currentUptime = formatTimeSinceDate(new Date(startTime));
    }
</script>

<Loadable state={metrics.data} loadable={metrics}>
    <PolledWidget title="System Metrics" pollFn={updateUptime} pollInterval={1000}>
        <div class="metrics">
            <MetricCard label="Uptime" value={currentUptime} --metric-value-size="1.1rem" />
        </div>
    </PolledWidget>
</Loadable>

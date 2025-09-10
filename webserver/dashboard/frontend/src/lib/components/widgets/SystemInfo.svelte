<script lang="ts">
    import { formatTimeSinceDate } from "$lib/utils/dates";
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import PolledWidget from "./base/PolledWidget.svelte";
    import { getContext } from "svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import Loadable from "../ui/Loadable.svelte";

    let metrics = getContext("metrics") as MetricsProvider;

    let currentUptime: string = $state("");

    function updateUptime() {
        if (!metrics) {
            return;
        }

        const startTime = metrics.data.system.startTime;
        currentUptime = formatTimeSinceDate(startTime);
    }
</script>

<PolledWidget title="System Metrics" pollFn={updateUptime} pollInterval={1000}>
    <Loadable loadable={metrics}>
        <div class="metrics">
            <MetricCard label="Uptime" value={currentUptime} --metric-value-size="1.1rem" />
        </div>
    </Loadable>
</PolledWidget>

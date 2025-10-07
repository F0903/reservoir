<script lang="ts">
    import { formatTimeSinceDate } from "$lib/utils/dates";
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { getContext } from "svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import Widget from "./base/Widget.svelte";
    import Poller from "./utils/Poller.svelte";

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

<Widget title="System Metrics">
    <Loadable state={metrics.data} loadable={metrics}>
        <Poller pollFn={updateUptime} pollInterval={1000}>
            <div class="metrics">
                <MetricCard label="Uptime" value={currentUptime} --metric-value-size="1.1rem" />
            </div>
        </Poller>
    </Loadable>
</Widget>

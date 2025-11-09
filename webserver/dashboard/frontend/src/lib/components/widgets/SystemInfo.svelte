<script lang="ts">
    import { formatTimeSinceDate } from "$lib/utils/dates";
    import { formatBytesToLargest } from "$lib/utils/bytestring";
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { getContext } from "svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import Widget from "./base/Widget.svelte";
    import Poller from "./utils/Poller.svelte";
    import type { Metrics } from "$lib/api/objects/metrics/metrics";

    const metrics = getContext("metrics") as MetricsProvider;

    let startTime: Date | null = $state(null);
    let uptime = $state("N/A");

    function updateUptime(data: Metrics) {
        if (!startTime && data.system.start_time) {
            startTime = new Date(data.system.start_time);
        }
        if (startTime) {
            uptime = formatTimeSinceDate(startTime);
        }
    }
</script>

<Widget title="System Info">
    <Loadable state={metrics.data} error={metrics.error}>
        {#snippet children(data)}
            <Poller pollFn={() => updateUptime(data)} pollInterval={1000}>
                <div class="metrics">
                    <MetricCard label="Uptime" value={uptime} />
                    <MetricCard label="Goroutines" value={data.system.num_goroutines} />
                    <MetricCard
                        label="Mem Allocated"
                        value={formatBytesToLargest(data.system.mem_alloc_bytes)}
                    />
                </div>
            </Poller>
        {/snippet}
    </Loadable>
</Widget>

<style>
    .metrics {
        display: flex;
        flex-direction: column;
        justify-content: space-around;
        gap: 0.5rem;
        height: 100%;
        width: 100%;
    }
</style>

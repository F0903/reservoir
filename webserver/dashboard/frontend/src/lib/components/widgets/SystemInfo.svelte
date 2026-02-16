<script lang="ts">
    import { formatTimeSinceDate } from "$lib/utils/dates";
    import { formatBytesToLargest } from "$lib/utils/bytestring";
    import MetricCard from "./utils/MetricCard.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import Widget from "./base/Widget.svelte";
    import Poller from "./utils/Poller.svelte";
    import type { Metrics } from "$lib/api/objects/metrics/metrics";
    import { getMetricsProvider } from "$lib/context";
    import { Clock, Cpu, MemoryStick, HardDrive, LayoutGrid } from "@lucide/svelte";

    const metrics = getMetricsProvider();

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
                    <MetricCard label="Uptime" value={uptime} icon={Clock} />
                    <MetricCard
                        label="Goroutines"
                        value={data.system.num_goroutines}
                        icon={LayoutGrid}
                    />
                    <MetricCard
                        label="Mem Allocated"
                        value={formatBytesToLargest(data.system.mem_alloc_bytes)}
                        icon={Cpu}
                    />
                    <MetricCard
                        label="Total Allocated"
                        value={formatBytesToLargest(data.system.mem_total_alloc_bytes)}
                        icon={MemoryStick}
                    />
                    <MetricCard
                        label="System Memory"
                        value={formatBytesToLargest(data.system.mem_sys_bytes)}
                        icon={HardDrive}
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
        gap: 0.35rem;
        height: 100%;
        width: 100%;
        box-sizing: border-box;
    }

    .metrics :global(.metric-card-wrapper) {
        flex: 1;
        min-height: 0;
    }
</style>

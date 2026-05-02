<script lang="ts">
    import { formatTimeSinceDate } from "$lib/utils/dates";
    import { formatBytesToLargest } from "$lib/utils/bytestring";
    import MetricCard from "./utils/MetricCard.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import Widget from "./base/Widget.svelte";
    import Poller from "./utils/Poller.svelte";
    import type { Metrics } from "$lib/api/objects/metrics/metrics";
    import { getMetricsProvider } from "$lib/context";
    import { Activity, Clock, Cpu } from "@lucide/svelte";
    import CapacityMetricCard from "./utils/CapacityMetricCard.svelte";

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

<Widget title="Process Info">
    <Loadable state={metrics.data} error={metrics.error}>
        {#snippet children(data)}
            {@const memoryPercent =
                data.system.mem_total_bytes > 0
                    ? Math.min(
                          100,
                          (data.system.mem_alloc_bytes / data.system.mem_total_bytes) * 100,
                      )
                    : 0}
            {@const footerItems = [
                { label: "Available", value: formatBytesToLargest(data.system.mem_total_bytes) },
            ]}
            <Poller pollFn={() => updateUptime(data)} pollInterval={1000}>
                <div class="process-panel">
                    <div class="summary-grid">
                        <MetricCard label="Uptime" value={uptime} icon={Clock} />
                        <MetricCard
                            label="Goroutines"
                            value={data.system.num_goroutines}
                            icon={Activity}
                        />
                        <MetricCard
                            label="Cores Available"
                            value={data.system.cores_available}
                            icon={Cpu}
                        />
                    </div>

                    <CapacityMetricCard
                        label="Mem Allocated"
                        value={formatBytesToLargest(data.system.mem_alloc_bytes)}
                        percent={memoryPercent}
                        progressLabel="Process memory allocation"
                        {footerItems}
                    />
                </div>
            </Poller>
        {/snippet}
    </Loadable>
</Widget>

<style>
    .process-panel {
        display: grid;
        grid-template-rows: auto minmax(0, 1fr);
        gap: 0.75rem;
        height: 100%;
        width: 100%;
        min-height: 0;
        box-sizing: border-box;
    }

    .summary-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
        gap: 0.6rem;
    }

    .summary-grid :global(.metric-card-wrapper) {
        --metric-padding: 0.55rem 0.65rem;
        --metric-border-radius: 8px;
        --metric-value-size: 0.95rem;
        --metric-label-size: 0.58rem;
        min-height: 3.6rem;
    }

    @media (max-width: 768px) {
        .process-panel {
            gap: 0.6rem;
        }
    }
</style>

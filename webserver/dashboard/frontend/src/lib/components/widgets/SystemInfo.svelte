<script lang="ts">
    import { formatTimeSinceDate } from "$lib/utils/dates";
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
        if (!startTime) {
            startTime = new Date(data.system.start_time);
            return;
        }

        uptime = formatTimeSinceDate(startTime);
    }
</script>

<Widget
    title="System Info"
    --widget-title-size="1rem"
    --widget-padding="1.5rem 0.75rem 0.75rem 0.75rem"
>
    <Loadable state={metrics.data} error={metrics.error}>
        {#snippet children(data)}
            <Poller pollFn={() => updateUptime(data)} pollInterval={1000}>
                <div class="metrics">
                    <MetricCard
                        label="Uptime"
                        value={uptime}
                        --metric-value-size="clamp(0.5rem, .9rem, 2rem)"
                        --metric-width="100%"
                        --metric-height="100%"
                    />
                </div>
            </Poller>
        {/snippet}
    </Loadable>
</Widget>

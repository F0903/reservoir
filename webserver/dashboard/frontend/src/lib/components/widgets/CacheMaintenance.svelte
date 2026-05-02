<script lang="ts">
    import Widget from "./base/Widget.svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import { formatBytesToLargest } from "$lib/utils/bytestring";
    import { getMetricsProvider } from "$lib/context";

    const metrics = getMetricsProvider();
</script>

<Widget title="Cache Maintenance">
    <Loadable state={metrics.data} error={metrics.error}>
        {#snippet children(data)}
            <div class="metrics-grid">
                <MetricCard label="Cleanup Runs" value={data.cache.cleanup_runs.toLocaleString()} />
                <MetricCard
                    label="Bytes Cleaned"
                    value={formatBytesToLargest(data.cache.bytes_cleaned)}
                />
                <MetricCard
                    label="Entries Evicted"
                    value={data.cache.cache_evictions.toLocaleString()}
                />
                <MetricCard
                    label="Cache Errors"
                    value={data.cache.cache_errors.toLocaleString()}
                    --metric-value-color="var(--log-error-color)"
                />
            </div>
        {/snippet}
    </Loadable>
</Widget>

<style>
    .metrics-grid {
        display: grid;
        grid-template-columns: repeat(2, minmax(0, 1fr));
        grid-template-rows: repeat(2, minmax(0, 1fr));
        gap: 0.75rem;
        height: 100%;
        width: 100%;
    }

    @media (max-width: 768px) {
        .metrics-grid {
            grid-template-columns: repeat(2, minmax(0, 1fr));
            grid-template-rows: auto;
        }
    }

    .metrics-grid :global(.metric-card-wrapper) {
        flex: 1;
        min-height: 0;
    }
</style>

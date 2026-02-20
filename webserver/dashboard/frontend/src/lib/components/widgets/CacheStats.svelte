<script lang="ts">
    import Widget from "./base/Widget.svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import { formatBytesToLargest } from "$lib/utils/bytestring";
    import { getMetricsProvider } from "$lib/context";

    const metrics = getMetricsProvider();
</script>

<Widget title="Cache Statistics">
    <Loadable state={metrics.data} error={metrics.error}>
        {#snippet children(data)}
            {@const totalBytesObserved = data.cache.bytes_cached + data.cache.bytes_cleaned}
            {@const fillPercent =
                totalBytesObserved > 0 ? (data.cache.bytes_cached / totalBytesObserved) * 100 : 0}
            <div class="metrics-grid">
                <MetricCard
                    label="Cache Entries"
                    value={data.cache.cache_entries.toLocaleString()}
                />
                <MetricCard
                    label="Cache Hits"
                    value={data.cache.cache_hits.toLocaleString()}
                    --metric-value-color="var(--success-color)"
                />
                <MetricCard label="Cache Misses" value={data.cache.cache_misses.toLocaleString()} />
                <MetricCard
                    label="Cache Errors"
                    value={data.cache.cache_errors.toLocaleString()}
                    --metric-value-color="var(--log-error-color)"
                />
                <MetricCard
                    label="Bytes Cached"
                    value={formatBytesToLargest(data.cache.bytes_cached)}
                />
                <MetricCard label="Cache Fill Level" value={`${fillPercent.toFixed(1)}%`} />
                <MetricCard label="Cleanup Runs" value={data.cache.cleanup_runs.toLocaleString()} />
                <MetricCard
                    label="Bytes Cleaned"
                    value={formatBytesToLargest(data.cache.bytes_cleaned)}
                />
                <MetricCard
                    label="Cache Evictions"
                    value={data.cache.cache_evictions.toLocaleString()}
                />
            </div>
        {/snippet}
    </Loadable>
</Widget>

<style>
    .metrics-grid {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        grid-template-rows: repeat(3, 1fr);
        gap: 0.5rem;
        height: 100%;
        width: 100%;
    }

    @media (max-width: 768px) {
        .metrics-grid {
            grid-template-columns: repeat(2, 1fr);
            grid-template-rows: auto;
        }
    }

    .metrics-grid :global(.metric-card-wrapper) {
        flex: 1;
        min-height: 0;
    }
</style>

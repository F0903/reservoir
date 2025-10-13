<script lang="ts">
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { getContext } from "svelte";
    import Widget from "./base/Widget.svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import { formatBytesToLargest } from "$lib/utils/bytestring";

    const metrics = getContext("metrics") as MetricsProvider;
</script>

<Widget title="Cache Statistics">
    <Loadable state={metrics.data} error={metrics.error}>
        {#snippet children(data)}
            <div class="metrics-grid">
                <MetricCard
                    label="Cache Entries"
                    value={data.cache.cache_entries.toLocaleString()}
                />
                <MetricCard label="Cache Hits" value={data.cache.cache_hits.toLocaleString()} />
                <MetricCard label="Cache Misses" value={data.cache.cache_misses.toLocaleString()} />
                <MetricCard label="Cache Errors" value={data.cache.cache_errors.toLocaleString()} />
                <MetricCard
                    label="Bytes Cached"
                    value={formatBytesToLargest(data.cache.bytes_cached)}
                />
                <MetricCard label="Cleanup Runs" value={data.cache.cleanup_runs.toLocaleString()} />
                <MetricCard
                    label="Bytes Cleaned"
                    value={formatBytesToLargest(data.cache.bytes_cleaned)}
                />
            </div>
        {/snippet}
    </Loadable>
</Widget>

<style>
    .metrics-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
        gap: 1rem;
    }
</style>

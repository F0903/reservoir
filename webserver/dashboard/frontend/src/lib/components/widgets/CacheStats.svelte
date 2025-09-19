<script lang="ts">
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { getContext } from "svelte";
    import Widget from "./base/Widget.svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import { formatBytesToLargest } from "$lib/utils/bytestring";

    let metrics = getContext("metrics") as MetricsProvider;
</script>

<Widget title="Cache Statistics">
    <Loadable loadable={metrics}>
        <div class="metrics-grid">
            <MetricCard
                label="Cache Entries"
                value={metrics.data.cache.cacheEntries.toLocaleString()}
            />
            <MetricCard label="Cache Hits" value={metrics.data.cache.cacheHits.toLocaleString()} />
            <MetricCard
                label="Cache Misses"
                value={metrics.data.cache.cacheMisses.toLocaleString()}
            />
            <MetricCard
                label="Cache Errors"
                value={metrics.data.cache.cacheErrors.toLocaleString()}
            />
            <MetricCard
                label="Bytes Cached"
                value={formatBytesToLargest(metrics.data.cache.bytesCached)}
            />
            <MetricCard
                label="Cleanup Runs"
                value={metrics.data.cache.cleanupRuns.toLocaleString()}
            />
            <MetricCard
                label="Bytes Cleaned"
                value={formatBytesToLargest(metrics.data.cache.bytesCleaned)}
            />
        </div>
    </Loadable>
</Widget>

<style>
    .metrics-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
        gap: 1rem;
    }
</style>

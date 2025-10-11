<script lang="ts">
    import type { MetricsProvider } from "$lib/providers/metric-providers.svelte";
    import { getContext } from "svelte";
    import Widget from "./base/Widget.svelte";
    import MetricCard from "./utils/MetricCard.svelte";
    import Loadable from "../ui/Loadable.svelte";
    import { formatBytesToLargest } from "$lib/utils/bytestring";

    const metrics = getContext("metrics") as MetricsProvider;

    let cacheEntries = $derived(metrics.data?.cache.cache_entries ?? 0);
    let totalCacheHits = $derived(metrics.data?.cache.cache_hits ?? 0);
    let totalCacheMisses = $derived(metrics.data?.cache.cache_misses ?? 0);
    let totalCacheErrors = $derived(metrics.data?.cache.cache_errors ?? 0);
    let totalBytesCached = $derived(metrics.data?.cache.bytes_cached ?? 0);
    let totalCleanupRuns = $derived(metrics.data?.cache.cleanup_runs ?? 0);
    let totalBytesCleaned = $derived(metrics.data?.cache.bytes_cleaned ?? 0);
</script>

<Widget title="Cache Statistics">
    <Loadable state={metrics.data} loadable={metrics}>
        <div class="metrics-grid">
            <MetricCard label="Cache Entries" value={cacheEntries.toLocaleString()} />
            <MetricCard label="Cache Hits" value={totalCacheHits.toLocaleString()} />
            <MetricCard label="Cache Misses" value={totalCacheMisses.toLocaleString()} />
            <MetricCard label="Cache Errors" value={totalCacheErrors.toLocaleString()} />
            <MetricCard label="Bytes Cached" value={formatBytesToLargest(totalBytesCached)} />
            <MetricCard label="Cleanup Runs" value={totalCleanupRuns.toLocaleString()} />
            <MetricCard label="Bytes Cleaned" value={formatBytesToLargest(totalBytesCleaned)} />
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

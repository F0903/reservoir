<script lang="ts">
    import type { MetricsProvider } from "$lib/providers/metrics.svelte";
    import { formatBytes } from "$lib/utils/format";
    import { getContext } from "svelte";
    import ErrorBox from "../ui/ErrorBox.svelte";
    import Widget from "./base/Widget.svelte";
    import MetricCard from "./utils/MetricCard.svelte";

    let metrics = getContext("metrics") as MetricsProvider;
</script>

<Widget title="Cache Statistics">
    {#if metrics.state.initializing}
        <p>Loading...</p>
    {:else if metrics.state.error}
        <ErrorBox><p>{metrics.state.error}</p></ErrorBox>
    {:else}
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
            <MetricCard label="Bytes Cached" value={formatBytes(metrics.data.cache.bytesCached)} />
            <MetricCard
                label="Cleanup Runs"
                value={metrics.data.cache.cleanupRuns.toLocaleString()}
            />
            <MetricCard
                label="Bytes Cleaned"
                value={formatBytes(metrics.data.cache.bytesCleaned)}
            />
        </div>
    {/if}
</Widget>

<style>
    .metrics-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
        gap: 1rem;
    }
</style>
